package main

import (
	"context"
	"embed"
	"flag"
	"os"
	"os/signal"
	"time"

	"dingtalk/internal/database"
	"dingtalk/internal/logger"
	"dingtalk/internal/server"
)

//go:embed dist
var distFS embed.FS

func main() {
	dbPath := flag.String("d", "", "database file path")
	port := flag.String("p", "8080", "server port")
	flag.Parse()

	if *dbPath == "" {
		logger.Fatal("database path is required")
	}

	db, err := database.MigrateToMemory(*dbPath)
	if err != nil {
		logger.Fatal("failed to migrate database: %v", err)
	}

	e := server.New(db, distFS)

	go func() {
		logger.Info("starting server on port %s", *port)
		if err := e.Start(":" + *port); err != nil {
			logger.Info("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Fatal("server shutdown failed: %v", err)
	}
	logger.Info("server exited")
}
