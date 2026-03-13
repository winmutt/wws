package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"wws/api/proto/auth"
	"wws/api/proto/organization"
	"wws/api/proto/user"
	"wws/api/proto/workspace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server holds the gRPC server and dependencies
type Server struct {
	auth.UnimplementedAuthServiceServer
	organization.UnimplementedOrganizationServiceServer
	user.UnimplementedUserServiceServer
	workspace.UnimplementedWorkspaceServiceServer

	httpServer *http.Server
}

// NewServer creates a new gRPC server
func NewServer(httpServer *http.Server) *Server {
	return &Server{
		httpServer: httpServer,
	}
}

// Start starts the gRPC server
func (s *Server) Start(port string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(s.loggingInterceptor),
	)

	// Register services
	auth.RegisterAuthServiceServer(grpcServer, s)
	organization.RegisterOrganizationServiceServer(grpcServer, s)
	user.RegisterUserServiceServer(grpcServer, s)
	workspace.RegisterWorkspaceServiceServer(grpcServer, s)

	// Register reflection service for debugging
	reflection.Register(grpcServer)

	log.Printf("gRPC server starting on port %s", port)
	return grpcServer.Serve(lis)
}

// loggingInterceptor logs gRPC calls
func (s *Server) loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)
	log.Printf("gRPC %s completed in %v", info.FullMethod, duration)
	return resp, err
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	// Graceful shutdown can be implemented here
}
