package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"wws/api/internal/compliance"
	"wws/api/internal/crypto"
	"wws/api/internal/db"
	"wws/api/internal/grpc"
	"wws/api/internal/handlers"
	"wws/api/internal/middleware"
	"wws/api/internal/routes"
	"wws/api/pkg"
)

func main() {
	config, err := pkg.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/wws.db"
	}
	db.Init(dbPath)

	// Initialize encryption for sensitive data at rest
	if err := crypto.InitEncryption(); err != nil {
		log.Printf("Warning: %v. Some sensitive data may not be encrypted.", err)
	} else {
		log.Println("Encryption initialized for data at rest")
	}

	// Initialize audit log handler
	handlers.AuditLogHandlerInstance = &handlers.AuditLogHandler{DB: db.DB}

	// Initialize OAuth state store
	handlers.InitOAuthStateStore()
	handlers.SetOAuthDB(db.DB)

	// Initialize quota handler
	handlers.QuotaHandlerInstance = &handlers.QuotaHandler{DB: db.DB}

	// Initialize API key handler
	handlers.APIKeyHandlerInstance = &handlers.APIKeyHandler{DB: db.DB}

	// Initialize compliance report generator and handler
	storagePath := os.Getenv("STORAGE_PATH")
	if storagePath == "" {
		storagePath = "./data"
	}
	auditLogPath := os.Getenv("AUDIT_LOG_PATH")
	if auditLogPath == "" {
		auditLogPath = "./data/audit_logs"
	}
	reportGenerator := compliance.NewComplianceReportGenerator(auditLogPath, storagePath)
	handlers.ComplianceHandlerInstance = handlers.NewComplianceHandler(reportGenerator)

	r := mux.NewRouter()

	routes.SetupRoutes(r)

	r.Use(middleware.CORSMiddleware(config.Server.CORS.Origins))
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)
	r.Use(middleware.RateLimitMiddleware(middleware.DefaultRateLimitConfig()))
	r.Use(middleware.AuditMiddleware(db.DB))

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = config.Server.Port
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090" // Default gRPC port
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: r,
	}

	// Start gRPC server in background
	grpcServer := grpc.NewServer(httpServer)
	go func() {
		if err := grpcServer.Start(grpcPort); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	log.Printf("HTTP server starting on port %s", httpPort)
	log.Printf("gRPC server starting on port %s", grpcPort)
	log.Printf("GitHub OAuth configured for: %s", config.GitHub.CallbackURL)
	log.Printf("CORS allowed origins: %v", config.Server.CORS.Origins)

	// Start HTTP server
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpServer.Shutdown(ctx)
	grpcServer.Stop()

	log.Println("Servers stopped")
}
