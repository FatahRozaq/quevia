package application

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/google/uuid"

	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/service/service"
)

type ServiceApplication struct {
	svc *service.CatalogService
}

func NewServiceApplication(svc *service.CatalogService) *ServiceApplication {
	return &ServiceApplication{svc: svc}
}

func AddRouteService(r *route.RouterGroup, svc *service.CatalogService, authMw app.HandlerFunc) {
	appHandler := NewServiceApplication(svc)

	categories := r.Group("/service-categories", authMw)
	categories.GET("", appHandler.ListCategories)
	categories.POST("", appHandler.CreateCategory)

	services := r.Group("/services", authMw)
	services.GET("", appHandler.ListServices)
	services.POST("", appHandler.CreateService)
}

func (h *ServiceApplication) ListCategories(ctx context.Context, c *app.RequestContext) {
	tenantID := middleware.GetTenantID(c)
	if tenantID == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	res, err := h.svc.ListCategories(ctx, *tenantID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *ServiceApplication) CreateCategory(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	var req payload.CreateCategoryPayload
	if err := c.BindAndValidate(&req); err != nil || req.Name == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.svc.CreateCategory(ctx, claims.TenantID, 1, req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusCreated, utils.H{"data": res})
}

func (h *ServiceApplication) ListServices(ctx context.Context, c *app.RequestContext) {
	tenantID := middleware.GetTenantID(c)
	if tenantID == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	branchGUIDStr := c.Query("branch_guid")
	branchGUID, err := uuid.Parse(branchGUIDStr)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_BRANCH_GUID"}})
		return
	}

	res, err := h.svc.ListServices(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *ServiceApplication) CreateService(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	var req payload.CreateServicePayload
	if err := c.BindAndValidate(&req); err != nil || req.Name == "" || req.QueuePrefix == "" || req.DurationMinutes <= 0 {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.svc.CreateService(ctx, claims.TenantID, 1, req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusCreated, utils.H{"data": res})
}
