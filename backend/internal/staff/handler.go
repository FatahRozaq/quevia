package staff

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
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *route.RouterGroup, authMw app.HandlerFunc) {
	staff := r.Group("/staff", authMw)
	staff.GET("", h.ListStaff)
	staff.POST("", h.CreateStaff)
	staff.PATCH("/:guid", h.UpdateStatus)
	staff.PUT("/:guid/services", h.AssignServices)
	staff.PUT("/:guid/schedules", h.SetSchedules)
}

func (h *Handler) ListStaff(ctx context.Context, c *app.RequestContext) {
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

	res, err := h.service.ListStaff(ctx, *tenantID, branchGUID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR"}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

type CreateStaffRequest struct {
	BranchGUID     uuid.UUID `json:"branch_guid"`
	UserGUID       uuid.UUID `json:"user_guid"`
	DisplayName    string    `json:"display_name"`
	Specialization string    `json:"specialization"`
}

func (h *Handler) CreateStaff(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{"error": utils.H{"code": "UNAUTHORIZED"}})
		return
	}

	var req CreateStaffRequest
	if err := c.BindAndValidate(&req); err != nil || req.DisplayName == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.service.CreateStaff(ctx, claims.TenantID, 1, req.BranchGUID, req.UserGUID, req.DisplayName, req.Specialization)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, utils.H{"data": res})
}

type UpdateStatusRequest struct {
	Status string `json:"status"` // ACTIVE or INACTIVE
}

func (h *Handler) UpdateStatus(ctx context.Context, c *app.RequestContext) {
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

	var req UpdateStatusRequest
	if err := c.BindAndValidate(&req); err != nil || req.Status == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	res, err := h.service.UpdateStaffStatus(ctx, claims.TenantID, 1, staffGUID, req.Status)
	if err != nil {
		// Enforce FR-SVC-06 / BR-19
		if errors.Is(err, ErrStaffHasActiveAppointments) {
			c.JSON(consts.StatusUnprocessableEntity, utils.H{
				"error": utils.H{
					"code":    "STAFF_HAS_ACTIVE_APPOINTMENTS",
					"message": err.Error(),
				},
			})
			return
		}
		if errors.Is(err, ErrStaffNotFound) {
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

type AssignServicesRequest struct {
	ServiceGUIDs []uuid.UUID `json:"service_guids"`
}

func (h *Handler) AssignServices(ctx context.Context, c *app.RequestContext) {
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

	var req AssignServicesRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	if err := h.service.AssignServices(ctx, claims.TenantID, 1, staffGUID, req.ServiceGUIDs); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"message": "services assigned successfully"})
}

func (h *Handler) SetSchedules(ctx context.Context, c *app.RequestContext) {
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

	var req []ScheduleInput
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	if err := h.service.SetStaffSchedules(ctx, claims.TenantID, 1, staffGUID, req); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"message": "staff schedules updated successfully"})
}
