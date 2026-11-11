package appointment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/google/uuid"

	"github.com/FatahRozaq/quevia/backend/internal/platform/idempotency"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
)

type Handler struct {
	service        *Service
	idempotencyMgr *idempotency.Manager
}

func NewHandler(service *Service, idempotencyMgr *idempotency.Manager) *Handler {
	return &Handler{
		service:        service,
		idempotencyMgr: idempotencyMgr,
	}
}

func (h *Handler) RegisterRoutes(r *route.RouterGroup, authMw app.HandlerFunc) {
	// Public & Authenticated Booking
	r.GET("/appointments/availability", h.GetAvailability)
	r.POST("/appointments", h.CreateAppointment)
	r.POST("/public/appointments/lookup", h.LookupBooking)
	r.POST("/appointments/:guid/cancel", h.CancelAppointment)
}

func (h *Handler) GetAvailability(ctx context.Context, c *app.RequestContext) {
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

	res, err := h.service.CalculateAvailability(ctx, AvailabilityInput{
		TenantID:    tID,
		BranchGUID:  branchGUID,
		ServiceGUID: serviceGUID,
		Date:        dateStr,
		StaffGUID:   staffGUIDPtr,
	})
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "AVAILABILITY_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

type CreateAppointmentRequest struct {
	BranchGUID  uuid.UUID     `json:"branch_guid"`
	ServiceGUID uuid.UUID     `json:"service_guid"`
	StaffGUID   *uuid.UUID    `json:"staff_guid,omitempty"`
	StartAt     time.Time     `json:"start_at"`
	Customer    CustomerInput `json:"customer"`
	Notes       string        `json:"notes,omitempty"`
}

func (h *Handler) CreateAppointment(ctx context.Context, c *app.RequestContext) {
	idempotencyKey := string(c.GetHeader("Idempotency-Key"))
	rawBody := c.Request.Body()

	var req CreateAppointmentRequest
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
		uid := claims.TenantID // if authenticated user
		_ = uid
	}

	cached, err := h.idempotencyMgr.Execute(
		ctx,
		tID,
		"POST /api/v1/appointments",
		idempotencyKey,
		rawBody,
		func(ctx context.Context) (int, interface{}, error) {
			res, err := h.service.CreateAppointment(ctx, CreateAppointmentInput{
				TenantID:    tID,
				BranchGUID:  req.BranchGUID,
				ServiceGUID: req.ServiceGUID,
				StaffGUID:   req.StaffGUID,
				StartAt:     req.StartAt,
				Customer:    req.Customer,
				Notes:       req.Notes,
				UserID:      userID,
			})
			if err != nil {
				return 0, nil, err
			}
			return http.StatusCreated, utils.H{"data": res}, nil
		},
	)

	if err != nil {
		if errors.Is(err, ErrSlotUnavailable) {
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

type LookupBookingRequest struct {
	BookingCode string `json:"booking_code"`
	Email       string `json:"email"`
}

func (h *Handler) LookupBooking(ctx context.Context, c *app.RequestContext) {
	var req LookupBookingRequest
	if err := c.BindAndValidate(&req); err != nil || req.BookingCode == "" || req.Email == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "VALIDATION_ERROR"}})
		return
	}

	tenantID := middleware.GetTenantID(c)
	tID := int64(1)
	if tenantID != nil {
		tID = *tenantID
	}

	res, err := h.service.LookupBooking(ctx, tID, req.BookingCode, req.Email)
	if err != nil {
		// Anti-enumeration: uniform 404
		c.JSON(consts.StatusNotFound, utils.H{
			"error": utils.H{"code": "BOOKING_NOT_FOUND", "message": "Booking not found"},
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"data": res})
}

type CancelAppointmentRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) CancelAppointment(ctx context.Context, c *app.RequestContext) {
	aptGUID, err := uuid.Parse(c.Param("guid"))
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": utils.H{"code": "INVALID_GUID"}})
		return
	}

	var req CancelAppointmentRequest
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

	res, err := h.service.CancelAppointment(ctx, tID, userID, aptGUID, req.Reason)
	if err != nil {
		if errors.Is(err, ErrChangeCutoffPassed) {
			c.JSON(consts.StatusUnprocessableEntity, utils.H{
				"error": utils.H{"code": "CHANGE_CUTOFF_PASSED", "message": err.Error()},
			})
			return
		}
		if errors.Is(err, ErrAppointmentNotFound) {
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
