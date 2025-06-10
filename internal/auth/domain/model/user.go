package model

import (
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system with multitenant support
type User struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID   string             `json:"userId" bson:"user_id"`
	Email    string             `json:"email" bson:"email"`
	Password string             `json:"-" bson:"password"` // Hidden from JSON

	// Profile information
	FirstName   string `json:"firstName" bson:"first_name"`
	LastName    string `json:"lastName" bson:"last_name"`
	DisplayName string `json:"displayName" bson:"display_name"`
	AvatarURL   string `json:"avatarUrl,omitempty" bson:"avatar_url,omitempty"`
	Phone       string `json:"phone,omitempty" bson:"phone,omitempty"`

	// Multitenant information
	TenantID         string   `json:"tenantId" bson:"tenant_id"`
	OrganizationID   string   `json:"organizationId,omitempty" bson:"organization_id,omitempty"`
	OrganizationRole string   `json:"organizationRole,omitempty" bson:"organization_role,omitempty"`
	Roles            []string `json:"roles" bson:"roles"`
	Permissions      []string `json:"permissions" bson:"permissions"`

	// Account status
	IsActive      bool       `json:"isActive" bson:"is_active"`
	IsVerified    bool       `json:"isVerified" bson:"is_verified"`
	LastLoginAt   *time.Time `json:"lastLoginAt,omitempty" bson:"last_login_at,omitempty"`
	LoginAttempts int        `json:"loginAttempts" bson:"login_attempts"`
	LockedUntil   *time.Time `json:"lockedUntil,omitempty" bson:"locked_until,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" bson:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" bson:"deleted_at,omitempty"`

	// Preferences
	Preferences map[string]interface{} `json:"preferences,omitempty" bson:"preferences,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (ve *ValidationError) Error() string {
	return ve.Field + ": " + ve.Message
}

// Common validation errors
var (
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidUserID   = errors.New("invalid user ID")
	ErrAccountLocked   = errors.New("account is locked")
	ErrAccountInactive = errors.New("account is inactive")
	ErrTenantMismatch  = errors.New("tenant mismatch")
)

// ValidateFields validates user fields for creation/update
func (u *User) ValidateFields() []ValidationError {
	var errs []ValidationError

	// Email validation
	if u.Email == "" {
		errs = append(errs, ValidationError{Field: "email", Message: "email is required"})
	} else if !isValidEmail(u.Email) {
		errs = append(errs, ValidationError{Field: "email", Message: "invalid email format"})
	}

	// Password validation (for creation)
	if u.Password != "" {
		if len(u.Password) < 8 {
			errs = append(errs, ValidationError{Field: "password", Message: "password must be at least 8 characters"})
		}
		if !hasUppercase(u.Password) {
			errs = append(errs, ValidationError{Field: "password", Message: "password must contain at least one uppercase letter"})
		}
		if !hasLowercase(u.Password) {
			errs = append(errs, ValidationError{Field: "password", Message: "password must contain at least one lowercase letter"})
		}
		if !hasDigit(u.Password) {
			errs = append(errs, ValidationError{Field: "password", Message: "password must contain at least one digit"})
		}
	}

	// First name validation
	if u.FirstName == "" {
		errs = append(errs, ValidationError{Field: "firstName", Message: "first name is required"})
	}

	// Last name validation
	if u.LastName == "" {
		errs = append(errs, ValidationError{Field: "lastName", Message: "last name is required"})
	}

	// Tenant ID validation
	if u.TenantID == "" {
		errs = append(errs, ValidationError{Field: "tenantId", Message: "tenant ID is required"})
	}

	return errs
}

// HashPassword hashes the user's password using bcrypt
func (u *User) HashPassword() error {
	if u.Password == "" {
		return ErrInvalidPassword
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Password = string(hashedBytes)
	return nil
}

// CheckPassword compares the provided password with the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// IsLocked returns true if the account is locked
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return u.LockedUntil.After(time.Now())
}

// CanLogin returns true if the user can login
func (u *User) CanLogin() bool {
	return u.IsActive && u.IsVerified && !u.IsLocked()
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// UpdateLastLogin updates the last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
	u.LoginAttempts = 0 // Reset failed attempts on successful login
}

// IncrementLoginAttempts increments the login attempts counter
func (u *User) IncrementLoginAttempts() {
	u.LoginAttempts++
	if u.LoginAttempts >= 5 { // Lock after 5 failed attempts
		lockUntil := time.Now().Add(30 * time.Minute) // Lock for 30 minutes
		u.LockedUntil = &lockUntil
	}
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.FirstName + " " + u.LastName
}

// BelongsToTenant checks if the user belongs to a specific tenant
func (u *User) BelongsToTenant(tenantID string) bool {
	return u.TenantID == tenantID
}

// BelongsToOrganization checks if the user belongs to a specific organization
func (u *User) BelongsToOrganization(organizationID string) bool {
	return u.OrganizationID == organizationID
}

// Helper functions

func isValidEmail(email string) bool {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}

func hasUppercase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func hasLowercase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func hasDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
