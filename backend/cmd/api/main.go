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

	"github.com/FatahRozaq/quevia/backend/internal/appointment"
	"github.com/FatahRozaq/quevia/backend/internal/auth"
	"github.com/FatahRozaq/quevia/backend/internal/branch"
	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/idempotency"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
	"github.com/FatahRozaq/quevia/backend/internal/service"
	"github.com/FatahRozaq/quevia/backend/internal/staff"
)

type AppHandlers struct {
	Auth        *auth.Handler
	Branch      *branch.Handler
	Service     *service.Handler
	Staff       *staff.Handler
	Appointment *appointment.Handler
}

func registerRoutes(
	h *server.Hertz,
	handlers *AppHandlers,
	authMw app.HandlerFunc,
) {
	h.GET("/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, utils.H{"status": "ok", "service": "quevia-api"})
	})

	if handlers != nil {
		v1 := h.Group("/api/v1")
		if handlers.Auth != nil {
			handlers.Auth.RegisterRoutes(v1, authMw)
		}
		if handlers.Branch != nil {
			handlers.Branch.RegisterRoutes(v1, authMw)
		}
		if handlers.Service != nil {
			handlers.Service.RegisterRoutes(v1, authMw)
		}
		if handlers.Staff != nil {
			handlers.Staff.RegisterRoutes(v1, authMw)
		}
		if handlers.Appointment != nil {
			handlers.Appointment.RegisterRoutes(v1, authMw)
		}
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

	var handlers *AppHandlers
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

		idempotencyMgr := idempotency.NewManager(pool, 24*time.Hour)

		authSvc := auth.NewService(pool, sessionStore, tokenManager, 14*24*time.Hour)
		branchSvc := branch.NewService(pool)
		serviceSvc := service.NewService(pool)
		staffSvc := staff.NewService(pool)
		aptSvc := appointment.NewService(pool)

		handlers = &AppHandlers{
			Auth:        auth.NewHandler(authSvc),
			Branch:      branch.NewHandler(branchSvc),
			Service:     service.NewHandler(serviceSvc),
			Staff:       staff.NewHandler(staffSvc),
			Appointment: appointment.NewHandler(aptSvc, idempotencyMgr),
		}
	}

	h := server.Default(server.WithHostPorts(":" + port))
	registerRoutes(h, handlers, authMw)
	h.Spin()
}
