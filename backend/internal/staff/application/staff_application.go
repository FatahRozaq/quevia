package application

import (
	"context"
	"errors"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/google/uuid"

	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/staff/service"
)

type StaffApplication struct {
	svc *service.StaffService
}

func NewStaffApplication(svc *service.StaffService) *StaffApplication {
	return &StaffApplication{svc: svc}
}

func AddRouteStaff(r *route.RouterGroup, svc *service.StaffService, authMw app.HandlerFunc) {
	appHandler := NewStaffApplication(svc)

	staff := r.Group("/staff", authMw)
	staff.GET("", appHandler.ListStaff)
	staff.POST("", appHandler.CreateStaff)
	staff.PATCH("/:guid", appHandler.UpdateStatus)
	staff.PUT("/:guid/services", appHandler.AssignServices)
	staff.PUT("/:guid/schedules", appHandler.SetSchedules)
}

func (h *StaffApplication) ListStaff(ctx context.Context, c *app.RequestContext) {
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

	res, err := h.svc.ListStaff(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *StaffApplication) CreateStaff(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	var req payload.CreateStaffPayload
	if err := c.BindAndValidate(&req); err != nil || req.DisplayName == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.svc.CreateStaff(ctx, claims.TenantID, 1, req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, utils.H{"data": res})
}

func (h *StaffApplication) UpdateStatus(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	staffGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	var req payload.UpdateStaffStatusPayload
	if err := c.BindAndValidate(&req); err != nil || req.Status == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.svc.UpdateStaffStatus(ctx, claims.TenantID, 1, staffGUID, req.Status)
	if err != nil {
		// Enforce FR-SVC-06 / BR-19
		if errors.Is(err, service.ErrStaffHasActiveAppointments) {
			c.JSON(consts.StatusUnprocessableEntity, utils.H{
				"error": utils.H{
					"code":    "STAFF_HAS_ACTIVE_APPOINTMENTS",
					"message": err.Error(),
				},
			})
			return
		}
		if errors.Is(err, service.ErrStaffNotFound) {
			c.JSON(consts.StatusNotFound, utils.H{
				"error": utils.H{"code": "STAFF_NOT_FOUND"},
			})
			return
		}
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *StaffApplication) AssignServices(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	staffGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	var req payload.AssignServicesPayload
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	if err := h.svc.AssignServices(ctx, claims.TenantID, 1, staffGUID, req.ServiceGUIDs); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"message": "services assigned successfully"})
}

func (h *StaffApplication) SetSchedules(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	staffGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	var req []payload.StaffSchedulePayload
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	if err := h.svc.SetStaffSchedules(ctx, claims.TenantID, 1, staffGUID, req); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"message": "staff schedules updated successfully"})
}
