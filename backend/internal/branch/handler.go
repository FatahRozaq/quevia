package branch

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
	branches := r.Group("/branches", authMw)
	branches.GET("/:guid/settings", h.GetSettings)
	branches.PUT("/:guid/settings", h.UpdateSettings)
	branches.GET("/:guid/business-hours", h.GetBusinessHours)
	branches.PUT("/:guid/business-hours", h.SetBusinessHours)

	counters := r.Group("/counters", authMw)
	counters.GET("", h.ListCounters)
	counters.POST("", h.CreateCounter)
}

func (h *Handler) GetSettings(ctx context.Context, c *app.RequestContext) {
	tenantID := middleware.GetTenantID(c)
	if tenantID == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	branchGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	res, err := h.service.GetSettings(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusNotFound, utils.H{"error": utils.H{"code": "BRANCH_NOT_FOUND"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *Handler) UpdateSettings(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	branchGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	var req SettingsInput
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.service.UpsertSettings(ctx, claims.TenantID, 1, branchGUID, req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *Handler) GetBusinessHours(ctx context.Context, c *app.RequestContext) {
	tenantID := middleware.GetTenantID(c)
	if tenantID == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	branchGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	res, err := h.service.ListBusinessHours(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *Handler) SetBusinessHours(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	branchGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	var req []BusinessHourInput
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	if err := h.service.SetBusinessHours(ctx, claims.TenantID, 1, branchGUID, req); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"message": "business hours updated successfully"})
}

func (h *Handler) ListCounters(ctx context.Context, c *app.RequestContext) {
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

	res, err := h.service.ListCounters(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

type CreateCounterRequest struct {
	BranchGUID   uuid.UUID `json:"branch_guid"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	DisplayOrder int16     `json:"display_order"`
}

func (h *Handler) CreateCounter(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	var req CreateCounterRequest
	if err := c.BindAndValidate(&req); err != nil || req.Code == "" || req.Name == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.service.CreateCounter(ctx, claims.TenantID, 1, req.BranchGUID, req.Code, req.Name, req.DisplayOrder)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusCreated, utils.H{"data": res})
}
