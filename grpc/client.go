package grpc

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/xakpro/cg-shared-libs/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// ClientConfig holds gRPC client configuration
type ClientConfig struct {
	Host               string        `yaml:"host"`
	Port               int           `yaml:"port"`
	Timeout            time.Duration `yaml:"timeout" env-default:"5s"`
	MaxRetries         int           `yaml:"max_retries" env-default:"3"`
	RetryWaitTime      time.Duration `yaml:"retry_wait_time" env-default:"100ms"`
	MaxRecvMsgSize     int           `yaml:"max_recv_msg_size" env-default:"4194304"`
	MaxSendMsgSize     int           `yaml:"max_send_msg_size" env-default:"4194304"`
	KeepAliveTime      time.Duration `yaml:"keep_alive_time" env-default:"30s"`
	KeepAliveTimeout   time.Duration `yaml:"keep_alive_timeout" env-default:"10s"`
	InitialWindowSize  int32         `yaml:"initial_window_size" env-default:"65536"`
	InitialConnWindow  int32         `yaml:"initial_conn_window" env-default:"65536"`
}

// Addr returns client target address
func (c *ClientConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Client wraps gRPC client connection
type Client struct {
	conn   *grpc.ClientConn
	config ClientConfig
}

// NewClient creates a new gRPC client connection
func NewClient(ctx context.Context, cfg ClientConfig, opts ...grpc.DialOption) (*Client, error) {
	defaultOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMsgSize),
		),
		grpc.WithChainUnaryInterceptor(
			clientLoggingInterceptor(),
			retryInterceptor(cfg.MaxRetries, cfg.RetryWaitTime),
		),
	}

	opts = append(defaultOpts, opts...)

	conn, err := grpc.DialContext(ctx, cfg.Addr(), opts...)
	if err != nil {
		return nil, fmt.Errorf("dial grpc: %w", err)
	}

	logger.Info("gRPC client connected",
		zap.String("addr", cfg.Addr()),
	)

	return &Client{
		conn:   conn,
		config: cfg,
	}, nil
}

// Conn returns the underlying connection
func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		logger.Info("gRPC client disconnected",
			zap.String("addr", c.config.Addr()),
		)
		return c.conn.Close()
	}
	return nil
}

// Client interceptors

func clientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		if code == codes.OK {
			logger.Debug("gRPC client call",
				zap.String("method", method),
				zap.Duration("duration", duration),
			)
		} else {
			logger.Warn("gRPC client call failed",
				zap.String("method", method),
				zap.Duration("duration", duration),
				zap.String("code", code.String()),
				zap.Error(err),
			)
		}

		return err
	}
}

func retryInterceptor(maxRetries int, waitTime time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var lastErr error

		for i := 0; i <= maxRetries; i++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			lastErr = err

			// Only retry on specific codes
			code := status.Code(err)
			if !isRetryable(code) {
				return err
			}

			if i < maxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(waitTime * time.Duration(i+1)):
					logger.Debug("retrying gRPC call",
						zap.String("method", method),
						zap.Int("attempt", i+2),
					)
				}
			}
		}

		return lastErr
	}
}

func isRetryable(code codes.Code) bool {
	switch code {
	case codes.Unavailable, codes.ResourceExhausted, codes.Aborted, codes.Internal:
		return true
	default:
		return false
	}
}

