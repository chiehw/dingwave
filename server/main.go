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

	"gorm.io/gorm"
)

//go:embed dist
var distFS embed.FS

func main() {
	dbPath := flag.String("d", "", "database file path")
	port := flag.String("p", "8080", "server port")
	keyUserID := flag.String("k", "", "解密密钥：V2 为目录 uid；V3 为 real_uid（见 log 中 real_uid，勿用目录名当 uid）")
	salt := flag.String("salt", "", "V3：user_config JSON 里的 salt/slt，与 -k 组合派生密钥")
	userConfig := flag.String("userconfig", "", "V3：user_config 文件路径（整文件 Base64），与 -k 的 real_uid 配合")
	outputPath := flag.String("o", "", "output path for decrypted database")
	mergedOut := flag.String("merged-out", "", "合并后的 SQLite 路径（含 conversations/messages 等表）；供离线脚本与 Agent 技能使用")
	exportOnly := flag.Bool("export-only", false, "仅写入 -merged-out 后退出（不启动 HTTP）；须与 -merged-out 同时使用")
	token := flag.String("token", "", "DingTalk account token for image download (optional)")
	flag.Parse()

	if *dbPath == "" {
		logger.Fatal("database path is required")
	}
	if *exportOnly && *mergedOut == "" {
		logger.Fatal("-export-only 需要同时指定 -merged-out")
	}

	finalDBPath := *dbPath

	if (*userConfig != "" || *salt != "") && *keyUserID == "" {
		logger.Fatal("V3 需要同时指定 -k（real_uid，可在 %%AppData%%\\\\Roaming\\\\DingTalk\\\\log 日志里搜 real_uid）")
	}

	if *keyUserID != "" {
		logger.Info("decrypting database...")
		var key []byte
		switch {
		case *userConfig != "":
			s, err := crypto.SaltFromUserConfigFile(*userConfig)
			if err != nil {
				logger.Fatal("读取 user_config: %v", err)
			}
			key = crypto.GenerateKeyV3(*keyUserID, s)
			logger.Info("V3：已从 user_config 读取 salt 并与 -k 派生密钥")
		case *salt != "":
			key = crypto.GenerateKeyV3(*keyUserID, *salt)
			logger.Info("V3：已使用 -salt 与 -k 派生密钥")
		default:
			key = crypto.GenerateKey(*keyUserID)
			logger.Info("V2：使用 MD5(-k) 派生密钥（若数据目录为 _v3 且解密失败，请改用 -userconfig/-salt）")
		}

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

	var db *gorm.DB
	var err error
	if *mergedOut != "" {
		db, err = database.MigrateToMergedFile(finalDBPath, *mergedOut)
	} else {
		db, err = database.MigrateToMemory(finalDBPath)
	}
	if err != nil {
		logger.Fatal("failed to migrate database: %v", err)
	}

	if *exportOnly {
		logger.Info("merged database written to %s, exiting (-export-only)", *mergedOut)
		return
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
