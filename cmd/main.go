package main

import (
	"RoyDental/cache"
	"RoyDental/config"
	"RoyDental/database"
	"RoyDental/routes"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	// Load configuration from config package
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Initialize the database
	db, err := database.InitDB(context.Background(), config.DBURL)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	// Initialize Redis
	if err := database.InitializeRedis(); err != nil {
		log.Fatalf("failed to initialize Redis client: %v", err)
	}

	// Initialize the cache utility
	cache, err := cache.NewCache()
	if err != nil {
		log.Fatalf("failed to initialize cache: %v", err)
	}

	// Pass the config to SetupRoutes
	handler := routes.SetupRoutes(cache, config, db)

	// Configure and start the server
	srv := &http.Server{
		Addr:           ":8930",
		Handler:        handler,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
		IdleTimeout:    30 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		log.Println("Starting server on :8900")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listenAndServe(): %v", err)
		}
	}()

	// Graceful shutdown handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Create a context with a timeout for shutdown
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	log.Println("Shutting down server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}

	wg.Wait() // Wait for all goroutines to finish before exiting
	log.Println("Server exited gracefully")
}

// loadConfig loads configuration from environment variables.
func loadConfig() (*config.AppConfig, error) {
	// Get the database URL
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, errors.New("missing DB_URL environment variable")
	}

	// Get the Redis URL
	redisAddress := os.Getenv("REDIS_URL")
	if redisAddress == "" {
		return nil, errors.New("missing REDIS_URL environment variable")
	}

	// Get the Bearer Token
	bearerToken := os.Getenv("BEARER_TOKEN")
	if bearerToken == "" {
		return nil, errors.New("missing BEARER_TOKEN environment variable")
	}

	// Returning the AppConfig with dynamic database name and other values
	return &config.AppConfig{
		DBURL:        dbURL,
		RedisAddress: redisAddress,
		BearerToken:  bearerToken,
	}, nil
}
