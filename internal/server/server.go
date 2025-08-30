package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/agentflow/agentflow/internal/security"
	"github.com/agentflow/agentflow/pkg/messaging"
	"github.com/gorilla/mux"
)

// Server represents the HTTP server for AgentFlow Control Plane
type Server struct {
	config         *Config
	logger         logging.Logger
	httpServer     *http.Server
	router         *mux.Router
	middleware     *MiddlewareStack
	tracing        *messaging.TracingMiddleware
	authenticator  security.Authenticator
	authMiddleware *security.AuthMiddleware
	authHandlers   *security.AuthHandlers
}

// New creates a new HTTP server instance
func New(config *Config, logger logging.Logger) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize tracing middleware
	tracingConfig := &messaging.TracingConfig{
		Enabled:      config.EnableTracing,
		OTLPEndpoint: config.TracingEndpoint,
		ServiceName:  config.ServiceName,
		SampleRate:   1.0,
	}

	tracing, err := messaging.NewTracingMiddleware(tracingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Create router
	router := mux.NewRouter()

	// Create authentication components
	authConfig := security.LoadAuthConfigFromEnv()
	authenticator, err := security.NewHybridAuthenticator(authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	authMiddleware := security.NewAuthMiddleware(authenticator, logger, authConfig)
	authHandlers := security.NewAuthHandlers(authenticator, logger)

	// Create middleware stack
	middlewareStack := NewMiddlewareStack(logger)
	middlewareStack.SetTracingMiddleware(tracing)
	middlewareStack.SetAuthMiddleware(authMiddleware)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.Port),
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	server := &Server{
		config:         config,
		logger:         logger,
		httpServer:     httpServer,
		router:         router,
		middleware:     middlewareStack,
		tracing:        tracing,
		authenticator:  authenticator,
		authMiddleware: authMiddleware,
		authHandlers:   authHandlers,
	}

	// Setup middleware chain in correct order
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	// Set the handler
	httpServer.Handler = server.middleware.Apply(router)

	return server, nil
}

// setupMiddleware configures the middleware chain in the correct order
func (s *Server) setupMiddleware() {
	// Middleware order is important - they are applied in reverse order
	// So the first middleware added will be the outermost (executed first)

	// 1. Recovery middleware (outermost - catches all panics)
	s.middleware.Use(s.middleware.RecoveryMiddleware())

	// 2. Logging middleware (logs all requests)
	s.middleware.Use(s.middleware.LoggingMiddleware())

	// 3. Tracing middleware (creates spans for requests)
	s.middleware.Use(s.middleware.TracingMiddleware())

	// 4. Authentication middleware (validates JWT tokens)
	s.middleware.Use(s.authMiddleware.Middleware())

	// 5. CORS middleware (handles cross-origin requests)
	s.middleware.Use(s.middleware.CORSMiddleware())
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// API v1 routes
	v1 := s.router.PathPrefix("/api/v1").Subrouter()

	// Health check endpoint (public)
	v1.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Authentication endpoints (public)
	v1.HandleFunc("/auth/token", s.authHandlers.HandleTokenIssue).Methods("POST")
	v1.HandleFunc("/auth/validate", s.authHandlers.HandleTokenValidate).Methods("POST")
	v1.HandleFunc("/auth/revoke", s.authHandlers.HandleTokenRevoke).Methods("POST")
	v1.HandleFunc("/auth/userinfo", s.authHandlers.HandleUserInfo).Methods("GET")

	// Protected endpoints for future implementation
	v1.HandleFunc("/workflows", s.handleWorkflows).Methods("GET", "POST")
	v1.HandleFunc("/workflows/{id}", s.handleWorkflow).Methods("GET", "PUT", "DELETE")
	v1.HandleFunc("/agents", s.handleAgents).Methods("GET", "POST")
	v1.HandleFunc("/agents/{id}", s.handleAgent).Methods("GET", "PUT", "DELETE")
	v1.HandleFunc("/tools", s.handleTools).Methods("GET", "POST")
	v1.HandleFunc("/tools/{id}", s.handleTool).Methods("GET", "PUT", "DELETE")

	// Root handler for API discovery (public)
	s.router.HandleFunc("/", s.handleRoot).Methods("GET")
	s.router.HandleFunc("/api", s.handleAPIRoot).Methods("GET")
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting AgentFlow Control Plane API server",
		logging.Int("port", s.config.Port),
		logging.Bool("tls_enabled", s.config.EnableTLS),
		logging.String("service_name", s.config.ServiceName),
	)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if s.config.EnableTLS {
			if s.config.TLSCertPath == "" || s.config.TLSKeyPath == "" {
				serverErr <- fmt.Errorf("TLS enabled but cert or key path not provided")
				return
			}
			serverErr <- s.httpServer.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath)
		} else {
			serverErr <- s.httpServer.ListenAndServe()
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	case <-ctx.Done():
		return s.Shutdown()
	}
}

// StartWithGracefulShutdown starts the server with graceful shutdown handling
func (s *Server) StartWithGracefulShutdown() error {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- s.Start(ctx)
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		s.logger.Info("Received shutdown signal", logging.String("signal", sig.String()))
		cancel()
		return <-serverErr
	case err := <-serverErr:
		return err
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down AgentFlow Control Plane API server")

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Error during server shutdown", err)
		return err
	}

	s.logger.Info("Server shutdown completed")
	return nil
}

// Handler implementations (placeholders for now)

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service": "AgentFlow Control Plane API",
		"version": "1.0.0",
		"status":  "running",
		"endpoints": map[string]string{
			"health": "/health",
			"api":    "/api/v1",
		},
	}
	s.writeJSONResponse(w, http.StatusOK, response)
}

func (s *Server) handleAPIRoot(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"version": "v1",
		"endpoints": map[string]string{
			"workflows": "/api/v1/workflows",
			"agents":    "/api/v1/agents",
			"tools":     "/api/v1/tools",
			"health":    "/api/v1/health",
			"auth":      "/api/v1/auth",
		},
		"auth_endpoints": map[string]string{
			"token":    "/api/v1/auth/token",
			"validate": "/api/v1/auth/validate",
			"revoke":   "/api/v1/auth/revoke",
			"userinfo": "/api/v1/auth/userinfo",
		},
	}
	s.writeJSONResponse(w, http.StatusOK, response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "agentflow-control-plane",
		"version":   "1.0.0",
	}
	s.writeJSONResponse(w, http.StatusOK, response)
}

// Placeholder handlers for future implementation
func (s *Server) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	s.writeNotImplemented(w, "Workflows endpoint not yet implemented")
}

func (s *Server) handleWorkflow(w http.ResponseWriter, r *http.Request) {
	s.writeNotImplemented(w, "Individual workflow endpoint not yet implemented")
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	s.writeNotImplemented(w, "Agents endpoint not yet implemented")
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	s.writeNotImplemented(w, "Individual agent endpoint not yet implemented")
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	s.writeNotImplemented(w, "Tools endpoint not yet implemented")
}

func (s *Server) handleTool(w http.ResponseWriter, r *http.Request) {
	s.writeNotImplemented(w, "Individual tool endpoint not yet implemented")
}

// Helper methods

func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Use proper JSON marshaling
	response := map[string]interface{}{
		"success": statusCode < 400,
		"data":    data,
	}

	if jsonBytes, err := json.Marshal(response); err == nil {
		w.Write(jsonBytes)
	} else {
		// Fallback to simple format if marshaling fails
		fmt.Fprintf(w, `{"success":%t,"error":"JSON marshaling failed"}`, statusCode < 400)
	}
}

func (s *Server) writeNotImplemented(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)

	fmt.Fprintf(w, `{"success":false,"error":{"code":"NOT_IMPLEMENTED","message":"%s"}}`, message)
}
