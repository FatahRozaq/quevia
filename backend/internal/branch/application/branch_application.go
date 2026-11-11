package application

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/google/uuid"

	"github.com/FatahRozaq/quevia/backend/internal/branch/service"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
)

type BranchApplication struct {
	svc *service.BranchService
}

func NewBranchApplication(svc *service.BranchService) *BranchApplication {
	return &BranchApplication{svc: svc}
}

func AddRouteBranch(r *route.RouterGroup, svc *service.BranchService, authMw app.HandlerFunc) {
	appHandler := NewBranchApplication(svc)

	branches := r.Group("/branches", authMw)
	branches.GET("/:guid/settings", appHandler.GetSettings)
	branches.PUT("/:guid/settings", appHandler.UpdateSettings)
	branches.GET("/:guid/business-hours", appHandler.GetBusinessHours)
	branches.PUT("/:guid/business-hours", appHandler.SetBusinessHours)

	counters := r.Group("/counters", authMw)
	counters.GET("", appHandler.ListCounters)
	counters.POST("", appHandler.CreateCounter)
}

func (h *BranchApplication) GetSettings(ctx context.Context, c *app.RequestContext) {
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

	res, err := h.svc.GetSettings(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusNotFound, utils.H{"error": utils.H{"code": "BRANCH_NOT_FOUND"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *BranchApplication) UpdateSettings(ctx context.Context, c *app.RequestContext) {
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

	var req payload.BranchSettingsPayload
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.svc.UpsertSettings(ctx, claims.TenantID, 1, branchGUID, req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *BranchApplication) GetBusinessHours(ctx context.Context, c *app.RequestContext) {
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

	res, err := h.svc.ListBusinessHours(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *BranchApplication) SetBusinessHours(ctx context.Context, c *app.RequestContext) {
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

	var req []payload.BusinessHourPayload
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	if err := h.svc.SetBusinessHours(ctx, claims.TenantID, 1, branchGUID, req); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"message": "business hours updated successfully"})
}

func (h *BranchApplication) ListCounters(ctx context.Context, c *app.RequestContext) {
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

	res, err := h.svc.ListCounters(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *BranchApplication) CreateCounter(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	var req payload.CreateCounterPayload
	if err := c.BindAndValidate(&req); err != nil || req.Code == "" || req.Name == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.svc.CreateCounter(ctx, claims.TenantID, 1, req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(http.StatusCreated, utils.H{"data": res})
}
