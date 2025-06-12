package model

import (
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OrganizationState represents the state of an organization
type OrganizationState string

const (
	OrganizationStateActive          OrganizationState = "ACTIVE"
	OrganizationStateSuspended       OrganizationState = "SUSPENDED"
	OrganizationStateDeleted         OrganizationState = "DELETED"
	OrganizationStatePendingDeletion OrganizationState = "PENDING_DELETION"
)

// Organization represents a Firestore organization (tenant)
type Organization struct {
	ID              primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	OrganizationID  string              `json:"organizationId" bson:"organization_id"`
	DisplayName     string              `json:"displayName" bson:"display_name"`
	Description     string              `json:"description,omitempty" bson:"description,omitempty"`
	BillingEmail    string              `json:"billingEmail" bson:"billing_email"`
	AdminEmails     []string            `json:"adminEmails,omitempty" bson:"admin_emails,omitempty"`
	DefaultLocation string              `json:"defaultLocation" bson:"default_location"`
	State           OrganizationState   `json:"state" bson:"state"`
	CreatedAt       time.Time           `json:"createdAt" bson:"created_at"`
	UpdatedAt       time.Time           `json:"updatedAt" bson:"updated_at"`
	DeletedAt       *time.Time          `json:"deletedAt,omitempty" bson:"deleted_at,omitempty"`
	ProjectCount    int                 `json:"projectCount" bson:"project_count"`
	Usage           *OrganizationUsage  `json:"usage,omitempty" bson:"usage,omitempty"`
	Quotas          *OrganizationQuotas `json:"quotas,omitempty" bson:"quotas,omitempty"`
}

// OrganizationUsage represents usage statistics for an organization
type OrganizationUsage struct {
	ProjectCount   int       `json:"projectCount" bson:"project_count"`
	DatabaseCount  int       `json:"databaseCount" bson:"database_count"`
	StorageBytes   int64     `json:"storageBytes" bson:"storage_bytes"`
	RequestCount   int64     `json:"requestCount" bson:"request_count"`
	BandwidthBytes int64     `json:"bandwidthBytes" bson:"bandwidth_bytes"`
	LastUpdated    time.Time `json:"lastUpdated" bson:"last_updated"`
}

// OrganizationQuotas represents quotas for an organization
type OrganizationQuotas struct {
	MaxProjects       int   `json:"maxProjects" bson:"max_projects"`
	MaxDatabases      int   `json:"maxDatabases" bson:"max_databases"`
	MaxStorageBytes   int64 `json:"maxStorageBytes" bson:"max_storage_bytes"`
	MaxRequestsPerDay int64 `json:"maxRequestsPerDay" bson:"max_requests_per_day"`
	MaxBandwidthBytes int64 `json:"maxBandwidthBytes" bson:"max_bandwidth_bytes"`
}

// Organization errors
var (
	ErrOrganizationNotFound  = errors.New("organization not found")
	ErrOrganizationExists    = errors.New("organization already exists")
	ErrInvalidOrganizationID = errors.New("invalid organization ID")
	ErrInvalidDisplayName    = errors.New("invalid display name")
	ErrInvalidBillingEmail   = errors.New("invalid billing email")
	ErrOrganizationDeleted   = errors.New("organization is deleted")
	ErrOrganizationSuspended = errors.New("organization is suspended")
)

// Validation regex patterns
var (
	organizationIDRegex = regexp.MustCompile(`^[a-z][-a-z0-9]{4,28}[a-z0-9]$`)
	emailRegex          = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// NewOrganization creates a new organization with validation
func NewOrganization(organizationID, displayName, billingEmail string) (*Organization, error) {
	if err := ValidateOrganizationID(organizationID); err != nil {
		return nil, err
	}

	if err := ValidateDisplayName(displayName); err != nil {
		return nil, err
	}

	if err := ValidateBillingEmail(billingEmail); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Organization{
		OrganizationID:  organizationID,
		DisplayName:     displayName,
		BillingEmail:    billingEmail,
		State:           OrganizationStateActive,
		CreatedAt:       now,
		UpdatedAt:       now,
		DefaultLocation: "us-central1",
		ProjectCount:    0,
		Usage: &OrganizationUsage{
			LastUpdated: now,
		},
		Quotas: &OrganizationQuotas{
			MaxProjects:       100,
			MaxDatabases:      500,
			MaxStorageBytes:   1073741824,  // 1GB
			MaxRequestsPerDay: 1000000,     // 1M requests
			MaxBandwidthBytes: 10737418240, // 10GB
		},
	}, nil
}

// ValidateOrganizationID validates organization ID format
func ValidateOrganizationID(organizationID string) error {
	if organizationID == "" {
		return ErrInvalidOrganizationID
	}
	if !organizationIDRegex.MatchString(organizationID) {
		return ErrInvalidOrganizationID
	}
	return nil
}

// ValidateDisplayName validates display name
func ValidateDisplayName(displayName string) error {
	if displayName == "" {
		return ErrInvalidDisplayName
	}
	if len(displayName) > 100 {
		return ErrInvalidDisplayName
	}
	return nil
}

// ValidateBillingEmail validates billing email format
func ValidateBillingEmail(email string) error {
	if email == "" {
		return ErrInvalidBillingEmail
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidBillingEmail
	}
	return nil
}

// IsActive returns true if the organization is active
func (o *Organization) IsActive() bool {
	return o.State == OrganizationStateActive
}

// IsDeleted returns true if the organization is deleted
func (o *Organization) IsDeleted() bool {
	return o.State == OrganizationStateDeleted || o.DeletedAt != nil
}

// IsSuspended returns true if the organization is suspended
func (o *Organization) IsSuspended() bool {
	return o.State == OrganizationStateSuspended
}

// CanCreateProject returns true if the organization can create a new project
func (o *Organization) CanCreateProject() bool {
	if !o.IsActive() {
		return false
	}
	if o.Quotas != nil && o.ProjectCount >= o.Quotas.MaxProjects {
		return false
	}
	return true
}

// UpdateUsage updates the organization usage statistics
func (o *Organization) UpdateUsage(usage *OrganizationUsage) {
	if usage != nil {
		o.Usage = usage
		o.Usage.LastUpdated = time.Now()
		o.UpdatedAt = time.Now()
	}
}

// MarkDeleted marks the organization as deleted
func (o *Organization) MarkDeleted() {
	o.State = OrganizationStateDeleted
	now := time.Now()
	o.DeletedAt = &now
	o.UpdatedAt = now
}

// Suspend suspends the organization
func (o *Organization) Suspend() {
	o.State = OrganizationStateSuspended
	o.UpdatedAt = time.Now()
}

// Activate activates the organization
func (o *Organization) Activate() {
	o.State = OrganizationStateActive
	o.UpdatedAt = time.Now()
}

// IsAdminEmail checks if the given email is an admin email for this organization
func (o *Organization) IsAdminEmail(email string) bool {
	for _, adminEmail := range o.AdminEmails {
		if adminEmail == email {
			return true
		}
	}
	return false
}

// AddAdminEmail adds an admin email to the organization
func (o *Organization) AddAdminEmail(email string) error {
	if err := ValidateBillingEmail(email); err != nil {
		return err
	}

	if !o.IsAdminEmail(email) {
		o.AdminEmails = append(o.AdminEmails, email)
		o.UpdatedAt = time.Now()
	}

	return nil
}

// RemoveAdminEmail removes an admin email from the organization
func (o *Organization) RemoveAdminEmail(email string) {
	for i, adminEmail := range o.AdminEmails {
		if adminEmail == email {
			o.AdminEmails = append(o.AdminEmails[:i], o.AdminEmails[i+1:]...)
			o.UpdatedAt = time.Now()
			break
		}
	}
}
