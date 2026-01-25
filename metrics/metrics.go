package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.com/xakpro/cg-shared-libs/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Metrics holds all Prometheus metrics for a service
type Metrics struct {
	serviceName string

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpErrorsTotal     *prometheus.CounterVec

	// gRPC metrics
	grpcRequestsTotal   *prometheus.CounterVec
	grpcRequestDuration *prometheus.HistogramVec
	grpcErrorsTotal     *prometheus.CounterVec
}

// New creates a new Metrics instance for a service
func New(serviceName string) *Metrics {
	return &Metrics{
		serviceName: serviceName,
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"service", "method", "endpoint", "status"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"service", "method", "endpoint"},
		),
		httpErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"service", "method", "endpoint", "error_type"},
		),
		grpcRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"service", "method", "status"},
		),
		grpcRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "grpc_request_duration_seconds",
				Help:    "gRPC request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"service", "method"},
		),
		grpcErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_errors_total",
				Help: "Total number of gRPC errors",
			},
			[]string{"service", "method", "error_code"},
		),
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	m.httpRequestsTotal.WithLabelValues(m.serviceName, method, endpoint, status).Inc()
	m.httpRequestDuration.WithLabelValues(m.serviceName, method, endpoint).Observe(duration.Seconds())

	if statusCode >= 400 {
		errorType := "client_error"
		if statusCode >= 500 {
			errorType = "server_error"
		}
		m.httpErrorsTotal.WithLabelValues(m.serviceName, method, endpoint, errorType).Inc()
	}
}

// RecordGRPCRequest records gRPC request metrics
func (m *Metrics) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.grpcRequestsTotal.WithLabelValues(m.serviceName, method, status).Inc()
	m.grpcRequestDuration.WithLabelValues(m.serviceName, method).Observe(duration.Seconds())

	if status != "OK" {
		m.grpcErrorsTotal.WithLabelValues(m.serviceName, method, status).Inc()
	}
}

// HTTPMetricsMiddleware wraps HTTP handler with metrics collection
func (m *Metrics) HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		endpoint := r.URL.Path
		method := r.Method

		m.RecordHTTPRequest(method, endpoint, rw.statusCode, duration)

		logger.Debug("HTTP request metrics",
			zap.String("service", m.serviceName),
			zap.String("method", method),
			zap.String("endpoint", endpoint),
			zap.Int("status", rw.statusCode),
			zap.Duration("duration", duration),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// GRPCMetricsInterceptor creates a gRPC interceptor for metrics
func (m *Metrics) GRPCMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		method := info.FullMethod

		statusCode := "OK"
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code().String()
			} else {
				statusCode = "Unknown"
			}
		}

		m.RecordGRPCRequest(method, statusCode, duration)

		logger.Debug("gRPC request metrics",
			zap.String("service", m.serviceName),
			zap.String("method", method),
			zap.String("status", statusCode),
			zap.Duration("duration", duration),
		)

		return resp, err
	}
}

// Handler returns the Prometheus metrics handler for /metrics endpoint
func Handler() http.Handler {
	return promhttp.Handler()
}
