package server

import (
	"dingtalk/internal/api"
	"dingtalk/internal/database"
	"dingtalk/internal/logger"
	"dingtalk/internal/service"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

func New(db *gorm.DB, distFS embed.FS) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			logger.Info("%s %s %d %s", c.Request().Method, c.Request().RequestURI, c.Response().Status, time.Since(start))
			return err
		}
	})
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	conversationService := service.NewConversationService(db)
	userService := service.NewUserService(db)
	messageService := service.NewMessageService(db)

	conversationHandler := api.NewConversationHandler(conversationService)
	userHandler := api.NewUserHandler(userService)
	messageHandler := api.NewMessageHandler(messageService, userService)

	apiGroup := e.Group("/api")
	apiGroup.GET("/conversations/home", conversationHandler.GetConversationsHome)
	apiGroup.GET("/conversations", conversationHandler.GetConversations)
	apiGroup.GET("/conversations/:cid/messages", messageHandler.GetConversationMessages)
	apiGroup.GET("/conversations/:cid/messages/search", messageHandler.SearchConversationMessages)
	apiGroup.GET("/messages/search", messageHandler.SearchMessagesGlobal)
	apiGroup.GET("/users", userHandler.GetUsers)
	apiGroup.GET("/users/search", userHandler.SearchUsers)

	var currentUser database.CurrentUser
	if err := db.First(&currentUser).Error; err == nil {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			staticPath := filepath.Join(homeDir, ".dingwave", fmt.Sprintf("%d", currentUser.ID))
			e.Static("/static", staticPath)
		}
	}

	distSubFS, _ := fs.Sub(distFS, "dist")
	e.GET("/*", echo.WrapHandler(http.FileServer(http.FS(distSubFS))))

	return e
}
