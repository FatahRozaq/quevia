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

	appointmentApp "github.com/FatahRozaq/quevia/backend/internal/appointment/application"
	appointmentSvc "github.com/FatahRozaq/quevia/backend/internal/appointment/service"
	authApp "github.com/FatahRozaq/quevia/backend/internal/auth/application"
	authSvc "github.com/FatahRozaq/quevia/backend/internal/auth/service"
	branchApp "github.com/FatahRozaq/quevia/backend/internal/branch/application"
	branchSvc "github.com/FatahRozaq/quevia/backend/internal/branch/service"
	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/idempotency"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
	serviceApp "github.com/FatahRozaq/quevia/backend/internal/service/application"
	serviceSvc "github.com/FatahRozaq/quevia/backend/internal/service/service"
	staffApp "github.com/FatahRozaq/quevia/backend/internal/staff/application"
	staffSvc "github.com/FatahRozaq/quevia/backend/internal/staff/service"
)

type AppServices struct {
	Auth        *authSvc.AuthService
	Branch      *branchSvc.BranchService
	Service     *serviceSvc.CatalogService
	Staff       *staffSvc.StaffService
	Appointment *appointmentSvc.AppointmentService
	Idempotency *idempotency.Manager
}

func registerRoutes(
	h *server.Hertz,
	services *AppServices,
	authMw app.HandlerFunc,
) {
	h.GET("/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, utils.H{"status": "ok", "service": "quevia-api"})
	})

	if services != nil {
		v1 := h.Group("/api/v1")
		if services.Auth != nil {
			authApp.AddRouteAuth(v1, services.Auth, authMw)
		}
		if services.Branch != nil {
			branchApp.AddRouteBranch(v1, services.Branch, authMw)
		}
		if services.Service != nil {
			serviceApp.AddRouteService(v1, services.Service, authMw)
		}
		if services.Staff != nil {
			staffApp.AddRouteStaff(v1, services.Staff, authMw)
		}
		if services.Appointment != nil {
			appointmentApp.AddRouteAppointment(v1, services.Appointment, services.Idempotency, authMw)
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

	var services *AppServices
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

		services = &AppServices{
			Auth:        authSvc.NewAuthService(pool, sessionStore, tokenManager, 14*24*time.Hour),
			Branch:      branchSvc.NewBranchService(pool),
			Service:     serviceSvc.NewCatalogService(pool),
			Staff:       staffSvc.NewStaffService(pool),
			Appointment: appointmentSvc.NewAppointmentService(pool),
			Idempotency: idempotencyMgr,
		}
	}

	h := server.Default(server.WithHostPorts(":" + port))
	registerRoutes(h, services, authMw)
	h.Spin()
}
