package payload

import (
	"strings"

	"github.com/google/uuid"
)

type RegisterTenantPayload struct {
	TenantName    string `json:"tenant_name"`
	TenantSlug    string `json:"tenant_slug"`
	BusinessType  string `json:"business_type"`
	ContactEmail  string `json:"contact_email"`
	BranchCode    string `json:"branch_code"`
	BranchName    string `json:"branch_name"`
	AdminFullName string `json:"admin_full_name"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	AdminPhone    string `json:"admin_phone"`
}

func (p *RegisterTenantPayload) Normalize() {
	p.TenantSlug = strings.ToLower(strings.TrimSpace(p.TenantSlug))
	p.AdminEmail = strings.ToLower(strings.TrimSpace(p.AdminEmail))
}

type RegisterTenantResponse struct {
	TenantGUID uuid.UUID `json:"tenant_guid"`
	TenantSlug string    `json:"tenant_slug"`
	BranchGUID uuid.UUID `json:"branch_guid"`
	UserGUID   uuid.UUID `json:"user_guid"`
}

type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (p *LoginPayload) Normalize() {
	p.Email = strings.ToLower(strings.TrimSpace(p.Email))
}

type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"-"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	GUID       uuid.UUID `json:"guid"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email"`
	TenantGUID uuid.UUID `json:"tenant_guid"`
	TenantSlug string    `json:"tenant_slug"`
	BranchGUID uuid.UUID `json:"branch_guid,omitempty"`
	Role       string    `json:"role"`
}
