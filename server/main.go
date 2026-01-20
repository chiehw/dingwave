package main

import (
	"context"
	"embed"
	"flag"
	"io"
	"os"
	"os/signal"
	"time"

	"dingtalk/internal/crypto"
	"dingtalk/internal/database"
	"dingtalk/internal/logger"
	"dingtalk/internal/server"
)

//go:embed dist
var distFS embed.FS

func main() {
	dbPath := flag.String("d", "", "database file path")
	port := flag.String("p", "8080", "server port")
	keyUserID := flag.String("k", "", "user ID for database decryption (optional)")
	outputPath := flag.String("o", "", "output path for decrypted database")
	token := flag.String("token", "", "DingTalk account token for image download (optional)")
	flag.Parse()

	if *dbPath == "" {
		logger.Fatal("database path is required")
	}

	finalDBPath := *dbPath

	if *keyUserID != "" {
		logger.Info("decrypting database...")
		key := crypto.GenerateKey(*keyUserID)

		tmpFile, err := os.CreateTemp("", "dingtalk-*.db")
		if err != nil {
			logger.Fatal("failed to create temp file: %v", err)
		}
		tempPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tempPath)

		if err := crypto.DecryptDatabase(*dbPath, tempPath, key); err != nil {
			logger.Fatal("failed to decrypt database: %v", err)
		}

		if err := database.ValidateDB(tempPath); err != nil {
			logger.Fatal("decryption failed: invalid database (wrong key?): %v", err)
		}

		logger.Info("decryption complete")

		if *outputPath != "" {
			if err := copyFile(tempPath, *outputPath); err != nil {
				logger.Fatal("failed to save decrypted database: %v", err)
			}
			logger.Info("decrypted database saved to %s", *outputPath)
		}

		finalDBPath = tempPath
	} else if *outputPath != "" {
		if err := copyFile(*dbPath, *outputPath); err != nil {
			logger.Fatal("failed to copy database: %v", err)
		}
		logger.Info("database copied to %s", *outputPath)
	}

	db, err := database.MigrateToMemory(finalDBPath)
	if err != nil {
		logger.Fatal("failed to migrate database: %v", err)
	}

	if *token != "" {
		if err := database.DownloadImages(db, *token); err != nil {
			logger.Error("failed to download images: %v", err)
		}
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
