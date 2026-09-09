package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/whu-ham/ham-gateway/internal/ham"
	"github.com/whu-ham/ham-gateway/internal/handlers"
	"github.com/whu-ham/ham-gateway/internal/middleware"
)

func main() {
	// Load environment variables
	baseURL := os.Getenv("HAM_API_BASE_URL")
	appID := os.Getenv("HAM_OPEN_APP_ID")
	appSecret := os.Getenv("HAM_OPEN_APP_SECRET")
	certPath := os.Getenv("HAM_GRPC_MTLS_CLIENT_CRT")
	keyPath := os.Getenv("HAM_GRPC_MTLS_CLIENT_KEY")
	authToken := os.Getenv("GATEWAY_AUTH_TOKEN")
	port := os.Getenv("PORT")

	// Set defaults
	if baseURL == "" {
		baseURL = "open-api.ham.nowcent.cn:4443"
	}
	if port == "" {
		port = "8080"
	}

	// Create HAM client
	hamClient, err := ham.NewClient(baseURL, appID, appSecret, certPath, keyPath)
	if err != nil {
		log.Fatalf("Failed to create HAM client: %v", err)
	}
	defer hamClient.Close()

	// Create handler
	hamHandler := handlers.NewHamHandler(hamClient)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s\n",
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
	}))
	r.Use(middleware.CORSMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes with authentication
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(authToken))
	{
		external := api.Group("/external")
		{
			ham := external.Group("/ham")
			{
				ham.GET("/course/search", hamHandler.SearchCourse)
				ham.GET("/score/stat", hamHandler.GetCourseStat)
				ham.GET("/score/by-course", hamHandler.GetCourseStatsByName)
			}
		}
	}

	// Start server
	log.Printf("Starting HAM Gateway on :%s", port)
	log.Printf("HAM API Base URL: %s", baseURL)
	if authToken != "" {
		log.Printf("Authentication enabled")
	} else {
		log.Printf("WARNING: Authentication disabled (no GATEWAY_AUTH_TOKEN set)")
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
