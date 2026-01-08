package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"gitlab.com/xakpro/cg-shared-libs/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	Host            string        `yaml:"host" env:"GRPC_HOST" env-default:"0.0.0.0"`
	Port            int           `yaml:"port" env:"GRPC_PORT" env-default:"50051"`
	MaxRecvMsgSize  int           `yaml:"max_recv_msg_size" env:"GRPC_MAX_RECV_MSG_SIZE" env-default:"4194304"` // 4MB
	MaxSendMsgSize  int           `yaml:"max_send_msg_size" env:"GRPC_MAX_SEND_MSG_SIZE" env-default:"4194304"` // 4MB
	ConnectionLimit int           `yaml:"connection_limit" env:"GRPC_CONN_LIMIT" env-default:"1000"`
	Timeout         time.Duration `yaml:"timeout" env:"GRPC_TIMEOUT" env-default:"30s"`
}

// Addr returns server address
func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Server wraps gRPC server
type Server struct {
	server   *grpc.Server
	listener net.Listener
	config   ServerConfig
}

// NewServer creates a new gRPC server
func NewServer(cfg ServerConfig, opts ...grpc.ServerOption) (*Server, error) {
	// Add default interceptors
	defaultOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(),
			loggingInterceptor(),
			timeoutInterceptor(cfg.Timeout),
		),
	}

	opts = append(defaultOpts, opts...)
	server := grpc.NewServer(opts...)

	return &Server{
		server: server,
		config: cfg,
	}, nil
}

// Server returns the underlying gRPC server
func (s *Server) Server() *grpc.Server {
	return s.server
}

// Start starts the gRPC server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.Addr())
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = listener

	logger.Info("gRPC server starting",
		zap.String("addr", s.config.Addr()),
	)

	return s.server.Serve(listener)
}

// Stop gracefully stops the server
func (s *Server) Stop() {
	logger.Info("gRPC server stopping")
	s.server.GracefulStop()
}

// Interceptors

func recoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC panic recovered",
					zap.Any("panic", r),
					zap.String("method", info.FullMethod),
				)
				err = status.Errorf(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

func loggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		// Log based on status
		if code == codes.OK {
			logger.Debug("gRPC request",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
			)
		} else {
			logger.Warn("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("code", code.String()),
				zap.Error(err),
			)
		}

		return resp, err
	}
}

func timeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(ctx, req)
	}
}

// GetMetadata extracts metadata from context
func GetMetadata(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// GetUserID extracts user_id from metadata
func GetUserID(ctx context.Context) int64 {
	val := GetMetadata(ctx, "x-user-id")
	if val == "" {
		return 0
	}
	var id int64
	fmt.Sscanf(val, "%d", &id)
	return id
}

// GetRequestID extracts request_id from metadata
func GetRequestID(ctx context.Context) string {
	return GetMetadata(ctx, "x-request-id")
}

// AuthInterceptorConfig holds auth interceptor configuration
type AuthInterceptorConfig struct {
	// SkipMethods - list of methods to skip auth (e.g., "/auth.AuthService/SendCode")
	SkipMethods []string
}

// JWTValidator interface for JWT validation
type JWTValidator interface {
	ValidateAccessToken(token string) (*JWTClaims, error)
}

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID   int64
	Phone    string
	DeviceID string
}

// AuthContextKey is the key for auth info in context
type authContextKey struct{}

// AuthInfo holds authenticated user info
type AuthInfo struct {
	UserID   int64
	Phone    string
	DeviceID string
}

// GetAuthInfo extracts auth info from context
func GetAuthInfo(ctx context.Context) (*AuthInfo, bool) {
	info, ok := ctx.Value(authContextKey{}).(*AuthInfo)
	return info, ok
}

// MustGetAuthInfo extracts auth info from context or panics
func MustGetAuthInfo(ctx context.Context) *AuthInfo {
	info, ok := GetAuthInfo(ctx)
	if !ok {
		panic("auth info not found in context")
	}
	return info
}

// AuthInterceptor creates authentication interceptor
func AuthInterceptor(validator JWTValidator, cfg AuthInterceptorConfig) grpc.UnaryServerInterceptor {
	skipMap := make(map[string]bool)
	for _, method := range cfg.SkipMethods {
		skipMap[method] = true
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Skip auth for certain methods
		if skipMap[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract token from metadata
		token := GetMetadata(ctx, "authorization")
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// Validate token
		claims, err := validator.ValidateAccessToken(token)
		if err != nil {
			logger.Warn("invalid token",
				zap.Error(err),
				zap.String("method", info.FullMethod),
			)
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Add auth info to context
		authInfo := &AuthInfo{
			UserID:   claims.UserID,
			Phone:    claims.Phone,
			DeviceID: claims.DeviceID,
		}
		ctx = context.WithValue(ctx, authContextKey{}, authInfo)

		// Also set user_id in metadata for backward compatibility
		ctx = metadata.AppendToOutgoingContext(ctx, "x-user-id", fmt.Sprintf("%d", claims.UserID))

		return handler(ctx, req)
	}
}

