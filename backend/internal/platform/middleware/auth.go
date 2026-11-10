package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
)

const (
	ContextClaimsKey   = "auth_claims"
	ContextTenantIDKey = "auth_tenant_id"
)

// AuthMiddleware validates JWT Bearer tokens on protected endpoints.
func AuthMiddleware(tm *token.TokenManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(consts.StatusUnauthorized, utils.H{
				"error": utils.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header missing or invalid format",
				},
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := tm.ValidateAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(consts.StatusUnauthorized, utils.H{
				"error": utils.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
			})
			return
		}

		c.Set(ContextClaimsKey, claims)
		c.Set(ContextTenantIDKey, claims.TenantID)
		c.Next(ctx)
	}
}

// GetClaims retrieves validated user claims from request context.
func GetClaims(c *app.RequestContext) *token.UserClaims {
	val, exists := c.Get(ContextClaimsKey)
	if !exists {
		return nil
	}
	claims, ok := val.(*token.UserClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetTenantID retrieves the current tenant ID from request context.
func GetTenantID(c *app.RequestContext) *int64 {
	val, exists := c.Get(ContextTenantIDKey)
	if !exists {
		return nil
	}
	id, ok := val.(int64)
	if !ok {
		return nil
	}
	return &id
}
