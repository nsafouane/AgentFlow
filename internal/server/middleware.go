package server

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Middleware represents an HTTP middleware function
type Middleware func(http.Handler) http.Handler

// MiddlewareStack manages the middleware chain
type MiddlewareStack struct {
	middlewares       []Middleware
	logger            logging.Logger
	tracer            trace.Tracer
	tracingMiddleware interface{} // Will hold *messaging.TracingMiddleware
}

// NewMiddlewareStack creates a new middleware stack
func NewMiddlewareStack(logger logging.Logger) *MiddlewareStack {
	return &MiddlewareStack{
		middlewares: make([]Middleware, 0),
		logger:      logger,
		tracer:      otel.Tracer("agentflow-control-plane"),
	}
}

// SetTracingMiddleware sets the messaging tracing middleware
func (ms *MiddlewareStack) SetTracingMiddleware(tracing interface{}) {
	ms.tracingMiddleware = tracing
}

// Use adds a middleware to the stack
func (ms *MiddlewareStack) Use(middleware Middleware) {
	ms.middlewares = append(ms.middlewares, middleware)
}

// Apply applies all middlewares to the given handler in order
func (ms *MiddlewareStack) Apply(handler http.Handler) http.Handler {
	// Apply middlewares in reverse order so they execute in the correct order
	for i := len(ms.middlewares) - 1; i >= 0; i-- {
		handler = ms.middlewares[i](handler)
	}
	return handler
}

// RecoveryMiddleware provides panic recovery with structured error responses
func (ms *MiddlewareStack) RecoveryMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic with stack trace
					stack := debug.Stack()
					ms.logger.Error("Panic recovered in HTTP handler",
						fmt.Errorf("panic: %v", err),
						logging.String("method", r.Method),
						logging.String("path", r.URL.Path),
						logging.String("remote_addr", r.RemoteAddr),
						logging.String("stack_trace", string(stack)),
					)

					// Return structured error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_SERVER_ERROR","message":"An internal server error occurred"}}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware provides structured request/response logging
func (ms *MiddlewareStack) LoggingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Extract correlation ID from headers or generate one
			correlationID := r.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				correlationID = generateCorrelationID()
			}

			// Add correlation ID to response headers
			rw.Header().Set("X-Correlation-ID", correlationID)

			// Create logger with request context
			requestLogger := ms.logger.WithFields(
				logging.String("correlation_id", correlationID),
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.String("remote_addr", r.RemoteAddr),
				logging.String("user_agent", r.UserAgent()),
			)

			// Add logger to request context
			ctx := context.WithValue(r.Context(), "logger", requestLogger)
			r = r.WithContext(ctx)

			// Log request start
			requestLogger.Info("HTTP request started")

			// Process request
			next.ServeHTTP(rw, r)

			// Log request completion
			duration := time.Since(start)
			requestLogger.Info("HTTP request completed",
				logging.Int("status_code", rw.statusCode),
				logging.Int("duration_ms", int(duration.Milliseconds())),
				logging.Int("response_size", rw.size),
			)
		})
	}
}

// TracingMiddleware provides OpenTelemetry tracing integration
func (ms *MiddlewareStack) TracingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Start a new span for the HTTP request
			ctx, span := ms.tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("http.host", r.Host),
					attribute.String("http.user_agent", r.UserAgent()),
					attribute.String("http.remote_addr", r.RemoteAddr),
				),
			)
			defer span.End()

			// Add trace context to request
			r = r.WithContext(ctx)

			// Create a response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(rw, r)

			// Add response attributes to span
			span.SetAttributes(
				attribute.Int("http.status_code", rw.statusCode),
				attribute.Int("http.response_size", rw.size),
			)

			// Set span status based on HTTP status code
			if rw.statusCode >= 400 {
				span.SetAttributes(attribute.Bool("error", true))
			}
		})
	}
}

// CORSMiddleware provides CORS headers for cross-origin requests
func (ms *MiddlewareStack) CORSMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Correlation-ID")
			w.Header().Set("Access-Control-Expose-Headers", "X-Correlation-ID")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// generateCorrelationID generates a unique correlation ID for request tracking
func generateCorrelationID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// GetLoggerFromContext extracts the logger from request context
func GetLoggerFromContext(ctx context.Context) logging.Logger {
	if logger, ok := ctx.Value("logger").(logging.Logger); ok {
		return logger
	}
	// Return default logger if not found in context
	return logging.NewLogger()
}
