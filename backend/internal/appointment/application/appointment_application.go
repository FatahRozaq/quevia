package application

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/google/uuid"

	"github.com/FatahRozaq/quevia/backend/internal/appointment/service"
	"github.com/FatahRozaq/quevia/backend/internal/platform/idempotency"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
)

type AppointmentApplication struct {
	svc            *service.AppointmentService
	idempotencyMgr *idempotency.Manager
}

func NewAppointmentApplication(svc *service.AppointmentService, idempotencyMgr *idempotency.Manager) *AppointmentApplication {
	return &AppointmentApplication{
		svc:            svc,
		idempotencyMgr: idempotencyMgr,
	}
}

func AddRouteAppointment(r *route.RouterGroup, svc *service.AppointmentService, idempotencyMgr *idempotency.Manager, authMw app.HandlerFunc) {
	appHandler := NewAppointmentApplication(svc, idempotencyMgr)

	r.GET("/appointments/availability", appHandler.GetAvailability)
	r.POST("/appointments", appHandler.CreateAppointment)
	r.POST("/public/appointments/lookup", appHandler.LookupBooking)
	r.POST("/appointments/:guid/cancel", appHandler.CancelAppointment)
}

func (h *AppointmentApplication) GetAvailability(ctx context.Context, c *app.RequestContext) {
	branchGUIDStr := c.Query("branch_guid")
	serviceGUIDStr := c.Query("service_guid")
	dateStr := c.Query("date")

	branchGUID, err := uuid.Parse(branchGUIDStr)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_BRANCH_GUID"}})
		return
	}

	serviceGUID, err := uuid.Parse(serviceGUIDStr)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_SERVICE_GUID"}})
		return
	}

	var staffGUIDPtr *uuid.UUID
	staffGUIDStr := c.Query("staff_guid")
	if staffGUIDStr != "" {
		sg, err := uuid.Parse(staffGUIDStr)
		if err == nil {
			staffGUIDPtr = &sg
		}
	}

	tenantID := middleware.GetTenantID(c)
	tID := int64(1)
	if tenantID != nil {
		tID = *tenantID
	}

	res, err := h.svc.CalculateAvailability(ctx, tID, branchGUID, serviceGUID, dateStr, staffGUIDPtr)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "AVAILABILITY_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *AppointmentApplication) CreateAppointment(ctx context.Context, c *app.RequestContext) {
	idempotencyKey := string(c.GetHeader("Idempotency-Key"))
	rawBody := c.Request.Body()

	var req payload.CreateAppointmentPayload
	if err := c.BindAndValidate(&req); err != nil || req.Customer.Email == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": utils.H{"code": "VALIDATION_ERROR", "message": "Invalid booking request"},
		})
		return
	}

	tenantID := middleware.GetTenantID(c)
	tID := int64(1)
	if tenantID != nil {
		tID = *tenantID
	}

	claims := middleware.GetClaims(c)
	var userID *int64
	if claims != nil {
		uid := claims.TenantID
		userID = &uid
	}

	cached, err := h.idempotencyMgr.Execute(
		ctx,
		tID,
		"POST /api/v1/appointments",
		idempotencyKey,
		rawBody,
		func(ctx context.Context) (int, interface{}, error) {
			res, err := h.svc.CreateAppointment(ctx, tID, userID, req)
			if err != nil {
				return 0, nil, err
			}
			return http.StatusCreated, utils.H{"data": res}, nil
		},
	)

	if err != nil {
		if errors.Is(err, service.ErrSlotUnavailable) {
			c.JSON(consts.StatusConflict, utils.H{
				"error": utils.H{
					"code":    "SLOT_UNAVAILABLE",
					"message": "Selected slot is no longer available",
				},
			})
			return
		}
		if errors.Is(err, idempotency.ErrIdempotencyKeyReused) {
			c.JSON(consts.StatusUnprocessableEntity, utils.H{
				"error": utils.H{
					"code":    "IDEMPOTENCY_KEY_REUSED",
					"message": err.Error(),
				},
			})
			return
		}
		if errors.Is(err, idempotency.ErrRequestInProgress) {
			c.JSON(consts.StatusConflict, utils.H{
				"error": utils.H{
					"code":    "REQUEST_IN_PROGRESS",
					"message": "A request with this Idempotency-Key is currently processing",
				},
			})
			return
		}
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": utils.H{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	if cached.Replayed {
		c.Response.Header.Set("Idempotency-Replayed", "true")
	}

	var parsedData interface{}
	_ = json.Unmarshal(cached.Body, &parsedData)
	c.JSON(cached.StatusCode, parsedData)
}

func (h *AppointmentApplication) LookupBooking(ctx context.Context, c *app.RequestContext) {
	var req payload.LookupBookingPayload
	if err := c.BindAndValidate(&req); err != nil || req.BookingCode == "" || req.Email == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	tenantID := middleware.GetTenantID(c)
	tID := int64(1)
	if tenantID != nil {
		tID = *tenantID
	}

	res, err := h.svc.LookupBooking(ctx, tID, req.BookingCode, req.Email)
	if err != nil {
		c.JSON(consts.StatusNotFound, utils.H{
			"error": utils.H{"code": "BOOKING_NOT_FOUND", "message": "Booking not found"},
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

func (h *AppointmentApplication) CancelAppointment(ctx context.Context, c *app.RequestContext) {
	aptGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	var req payload.CancelAppointmentPayload
	_ = c.BindAndValidate(&req)

	tenantID := middleware.GetTenantID(c)
	tID := int64(1)
	if tenantID != nil {
		tID = *tenantID
	}

	claims := middleware.GetClaims(c)
	var userID *int64
	if claims != nil {
		uid := claims.TenantID
		userID = &uid
	}

	res, err := h.svc.CancelAppointment(ctx, tID, userID, aptGUID, req.Reason)
	if err != nil {
		if errors.Is(err, service.ErrChangeCutoffPassed) {
			c.JSON(consts.StatusUnprocessableEntity, utils.H{
				"error": utils.H{"code": "CHANGE_CUTOFF_PASSED", "message": err.Error()},
			})
			return
		}
		if errors.Is(err, service.ErrAppointmentNotFound) {
			c.JSON(consts.StatusNotFound, utils.H{
				"error": utils.H{"code": "NOT_FOUND"},
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
