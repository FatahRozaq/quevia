package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/FatahRozaq/quevia/backend/internal/auth"
	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
)

func registerRoutes(
	h *server.Hertz,
	authHandler *auth.Handler,
	authMw app.HandlerFunc,
) {
	h.GET("/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, utils.H{"status": "ok", "service": "quevia-api"})
	})

	if authHandler != nil {
		v1 := h.Group("/api/v1")
		authHandler.RegisterRoutes(v1, authMw)
	}
}

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	jwtKey := os.Getenv("JWT_SIGNING_KEY")
	if jwtKey == "" {
		jwtKey = "default-quevia-dev-secret-key-min-32-chars!"
	}

	tokenManager := token.NewTokenManager(jwtKey, 15*time.Minute)
	authMw := middleware.AuthMiddleware(tokenManager)

	var authHandler *auth.Handler
	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")

	if dbURL != "" && redisURL != "" {
		ctx := context.Background()
		pool, err := database.NewPool(ctx, dbURL)
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		defer pool.Close()

		sessionStore, err := cache.NewSessionStore(redisURL)
		if err != nil {
			log.Fatalf("failed to connect to redis: %v", err)
		}

		authService := auth.NewService(pool, sessionStore, tokenManager, 14*24*time.Hour)
		authHandler = auth.NewHandler(authService)
	}

	h := server.Default(server.WithHostPorts(":" + port))
	registerRoutes(h, authHandler, authMw)
	h.Spin()
}
