package gin

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/tangthinker/controlman/internal/daemon"
)

func RegisterRoutes(router *gin.Engine, daemon *daemon.Daemon, authParams *AuthParams) {
	authMiddleware := MakeAuthMiddleware(authParams)
	controller := NewController(daemon)

	// 如果系统用户为root
	prefix := "./"
	if os.Getuid() == 0 {
		prefix = "/root/.controlman"
	}
	router.Static("/assets", prefix+"/static")
	router.StaticFile("/", prefix+"/static/login.html")
	router.StaticFile("/dashboard", prefix+"/static/index.html")
	router.StaticFile("/info", prefix+"/static/info.html")

	router.POST("/command", authMiddleware, controller.Command)
}

func StartServer(daemon *daemon.Daemon, authParams *AuthParams) {
	go func() {
		log.Println("Starting server on port 1984")

		homeDir, err := os.UserHomeDir()
		if err == nil {
			logFilePath := filepath.Join(homeDir, ".controlman", "controlman-api.log")
			logFile, openErr := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if openErr == nil {
				// Remove os.Stdout/Stderr to stop writing logs to terminal
				gin.DefaultWriter = logFile
				gin.DefaultErrorWriter = logFile
			} else {
				log.Printf("Failed to open log file: %v, using stdout only", openErr)
			}
		} else {
			log.Printf("Failed to get home directory: %v, using stdout only", err)
		}

		router := gin.Default()
		RegisterRoutes(router, daemon, authParams)
		if runErr := router.Run(":1984"); runErr != nil {
			log.Printf("Failed to start server: %v", runErr)
		}
	}()
}
