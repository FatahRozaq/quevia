package application

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

	"github.com/FatahRozaq/quevia/backend/internal/auth/service"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
)

type AuthApplication struct {
	svc *service.AuthService
}

func NewAuthApplication(svc *service.AuthService) *AuthApplication {
	return &AuthApplication{svc: svc}
}

func AddRouteAuth(r *route.RouterGroup, svc *service.AuthService, authMw app.HandlerFunc) {
	appHandler := NewAuthApplication(svc)
	r.POST("/tenants", appHandler.RegisterTenant)
	r.POST("/auth/login", appHandler.Login)
	r.GET("/me", authMw, appHandler.Me)
}

func (h *AuthApplication) RegisterTenant(ctx context.Context, c *app.RequestContext) {
	var req payload.RegisterTenantPayload
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": utils.H{"code": "VALIDATION_ERROR", "message": "Invalid request payload"},
		})
		return
	}

	if req.TenantName == "" || req.TenantSlug == "" || req.AdminEmail == "" || req.AdminPassword == "" || req.BranchCode == "" || req.BranchName == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": utils.H{"code": "VALIDATION_ERROR", "message": "All required fields must be provided"},
		})
		return
	}

	result, err := h.svc.RegisterTenant(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrTenantSlugExists) {
			c.JSON(consts.StatusConflict, utils.H{
				"error": utils.H{"code": "SLUG_ALREADY_EXISTS", "message": "Tenant slug already taken"},
			})
			return
		}
		if errors.Is(err, service.ErrUserEmailExists) {
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

func (h *AuthApplication) Login(ctx context.Context, c *app.RequestContext) {
	var req payload.LoginPayload
	if err := c.BindAndValidate(&req); err != nil || strings.TrimSpace(req.Email) == "" || req.Password == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": utils.H{"code": "VALIDATION_ERROR", "message": "Email and password are required"},
		})
		return
	}

	result, err := h.svc.Login(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrNoActiveMembership) {
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
			"user":         result.User,
		},
	})
}

func (h *AuthApplication) Me(ctx context.Context, c *app.RequestContext) {
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
