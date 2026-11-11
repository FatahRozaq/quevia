package service

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/google/uuid"

	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *route.RouterGroup, authMw app.HandlerFunc) {
	categories := r.Group("/service-categories", authMw)
	categories.GET("", h.ListCategories)
	categories.POST("", h.CreateCategory)

	services := r.Group("/services", authMw)
	services.GET("", h.ListServices)
	services.POST("", h.CreateService)
}

func (h *Handler) ListCategories(ctx context.Context, c *app.RequestContext) {
	tenantID := middleware.GetTenantID(c)
	if tenantID == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	res, err := h.service.ListCategories(ctx, *tenantID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

type CreateCategoryRequest struct {
	Name      string `json:"name"`
	SortOrder int16  `json:"sort_order"`
}

func (h *Handler) CreateCategory(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	var req CreateCategoryRequest
	if err := c.BindAndValidate(&req); err != nil || req.Name == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.service.CreateCategory(ctx, claims.TenantID, 1, req.Name, req.SortOrder)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusCreated, utils.H{"data": res})
}

func (h *Handler) ListServices(ctx context.Context, c *app.RequestContext) {
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

	res, err := h.service.ListServices(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *Handler) CreateService(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	var req CreateServiceInput
	if err := c.BindAndValidate(&req); err != nil || req.Name == "" || req.QueuePrefix == "" || req.DurationMinutes <= 0 {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.service.CreateService(ctx, claims.TenantID, 1, req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusCreated, utils.H{"data": res})
}
