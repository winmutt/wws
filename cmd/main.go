package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

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

	r := mux.NewRouter()

	routes.SetupRoutes(r)

	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)

	port := os.Getenv("PORT")
	if port == "" {
		port = config.Server.Port
	}

	log.Printf("Starting server on port %s", port)
	log.Printf("GitHub OAuth configured for: %s", config.GitHub.CallbackURL)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}
