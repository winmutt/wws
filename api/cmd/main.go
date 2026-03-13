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

	"wws/api/internal/grpc"
	"wws/api/internal/middleware"
	"wws/api/internal/routes"
	"wws/api/pkg"
)

func main() {
	config, err := pkg.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	r := mux.NewRouter()

	routes.SetupRoutes(r)

	r.Use(middleware.CORSMiddleware(config.Server.CORS.Origins))
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)

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
