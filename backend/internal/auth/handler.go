package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers auth endpoints to Hertz router.
func (h *Handler) RegisterRoutes(r *route.RouterGroup, authMw app.HandlerFunc) {
	r.POST("/tenants", h.RegisterTenant)
	r.POST("/auth/login", h.Login)
	r.GET("/me", authMw, h.Me)
}

// RegisterTenant handles tenant registration request.
func (h *Handler) RegisterTenant(ctx context.Context, c *app.RequestContext) {
	var req RegisterTenantInput
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": utils.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request payload",
			},
		})
		return
	}

	if req.TenantName == "" || req.TenantSlug == "" || req.AdminEmail == "" || req.AdminPassword == "" || req.BranchCode == "" || req.BranchName == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": utils.H{
				"code":    "VALIDATION_ERROR",
				"message": "All required fields must be provided",
			},
		})
		return
	}

	result, err := h.service.RegisterTenant(ctx, req)
	if err != nil {
		if errors.Is(err, ErrTenantSlugExists) {
			c.JSON(consts.StatusConflict, utils.H{
				"error": utils.H{"code": "SLUG_ALREADY_EXISTS", "message": "Tenant slug already taken"},
			})
			return
		}
		if errors.Is(err, ErrUserEmailExists) {
			c.JSON(consts.StatusConflict, utils.H{
				"error": utils.H{"code": "EMAIL_ALREADY_EXISTS", "message": "Admin email already registered"},
			})
			return
		}
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": utils.H{"code": "INTERNAL_ERROR", "message": "Failed to register tenant"},
		})
		return
	}

	c.JSON(consts.StatusCreated, utils.H{
		"data": result,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates user and sets refresh token in httpOnly cookie.
func (h *Handler) Login(ctx context.Context, c *app.RequestContext) {
	var req LoginRequest
	if err := c.BindAndValidate(&req); err != nil || strings.TrimSpace(req.Email) == "" || req.Password == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": utils.H{"code": "VALIDATION_ERROR", "message": "Email and password are required"},
		})
		return
	}

	result, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrNoActiveMembership) {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"error": utils.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password"},
			})
			return
		}
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": utils.H{"code": "INTERNAL_ERROR", "message": "Failed to process login"},
		})
		return
	}

	// Set refresh token in httpOnly cookie
	cookie := protocol.AcquireCookie()
	cookie.SetKey("refresh_token")
	cookie.SetValue(result.RefreshToken)
	cookie.SetPath("/api/v1/auth")
	cookie.SetHTTPOnly(true)
	cookie.SetSameSite(protocol.CookieSameSiteLaxMode)
	cookie.SetMaxAge(14 * 24 * 3600)
	c.Response.Header.SetCookie(cookie)
	protocol.ReleaseCookie(cookie)

	c.JSON(consts.StatusOK, utils.H{
		"data": utils.H{
			"access_token": result.AccessToken,
			"user": utils.H{
				"guid":        result.UserGUID,
				"full_name":   result.FullName,
				"email":       result.Email,
				"tenant_guid": result.TenantGUID,
				"tenant_slug": result.TenantSlug,
				"branch_guid": result.BranchGUID,
				"role":        result.Role,
			},
		},
	})
}

// Me returns the authenticated user's claims from the token context.
func (h *Handler) Me(ctx context.Context, c *app.RequestContext) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(consts.StatusUnauthorized, utils.H{
			"error": utils.H{"code": "UNAUTHORIZED", "message": "Missing credentials"},
		})
		return
	}

	c.JSON(http.StatusOK, utils.H{
		"data": utils.H{
			"user_guid":   claims.UserGUID,
			"tenant_guid": claims.TenantGUID,
			"role":        claims.Role,
			"branch_guid": claims.BranchGUID,
		},
	})
}
