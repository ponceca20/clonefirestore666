package model

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Organization represents a Firestore organization (tenant)
// This is the top-level entity in the Firestore hierarchy:
// Organization → Projects → Databases → Documents
type Organization struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	OrganizationID string             `json:"organizationId" bson:"organization_id"` // Unique identifier

	// Basic information
	DisplayName string `json:"displayName" bson:"display_name"`
	Description string `json:"description,omitempty" bson:"description,omitempty"`

	// Contact and billing
	BillingEmail string   `json:"billingEmail" bson:"billing_email"`
	AdminEmails  []string `json:"adminEmails" bson:"admin_emails"`
	Domain       string   `json:"domain,omitempty" bson:"domain,omitempty"`

	// Geographic and compliance
	DefaultLocation string          `json:"defaultLocation" bson:"default_location"` // e.g., "us-central1"
	DataResidency   []string        `json:"dataResidency,omitempty" bson:"data_residency,omitempty"`
	ComplianceInfo  *ComplianceInfo `json:"complianceInfo,omitempty" bson:"compliance_info,omitempty"`

	// Resource management
	Quotas  *OrganizationQuotas `json:"quotas,omitempty" bson:"quotas,omitempty"`
	Usage   *OrganizationUsage  `json:"usage,omitempty" bson:"usage,omitempty"`
	Billing *BillingInfo        `json:"billing,omitempty" bson:"billing,omitempty"`

	// State management
	State     OrganizationState `json:"state" bson:"state"`
	CreatedAt time.Time         `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time         `json:"updatedAt" bson:"updated_at"`
	DeletedAt *time.Time        `json:"deletedAt,omitempty" bson:"deleted_at,omitempty"`

	// Access control
	IamPolicy *IAMPolicy `json:"iamPolicy,omitempty" bson:"iam_policy,omitempty"`

	// Projects count (denormalized for performance)
	ProjectCount int `json:"projectCount" bson:"project_count"`
}

// OrganizationState represents the current state of an organization
type OrganizationState string

const (
	OrganizationStateActive       OrganizationState = "ACTIVE"
	OrganizationStateSuspended    OrganizationState = "SUSPENDED"
	OrganizationStateDeleted      OrganizationState = "DELETE_REQUESTED"
	OrganizationStatePendingSetup OrganizationState = "PENDING_SETUP"
)

// ComplianceInfo holds compliance-related information
type ComplianceInfo struct {
	GDPRCompliant      bool     `json:"gdprCompliant" bson:"gdpr_compliant"`
	HIPAACompliant     bool     `json:"hipaaCompliant" bson:"hipaa_compliant"`
	SOX                bool     `json:"sox" bson:"sox"`
	DataClassification string   `json:"dataClassification" bson:"data_classification"`
	RetentionPolicies  []string `json:"retentionPolicies,omitempty" bson:"retention_policies,omitempty"`
}

// OrganizationQuotas defines resource limits for the organization
type OrganizationQuotas struct {
	MaxProjects      int   `json:"maxProjects" bson:"max_projects"`
	MaxDatabases     int   `json:"maxDatabases" bson:"max_databases"`
	MaxStorageGB     int64 `json:"maxStorageGB" bson:"max_storage_gb"`
	MaxDocuments     int64 `json:"maxDocuments" bson:"max_documents"`
	MaxReadsPerDay   int64 `json:"maxReadsPerDay" bson:"max_reads_per_day"`
	MaxWritesPerDay  int64 `json:"maxWritesPerDay" bson:"max_writes_per_day"`
	MaxDeletesPerDay int64 `json:"maxDeletesPerDay" bson:"max_deletes_per_day"`
}

// OrganizationUsage tracks current resource usage
type OrganizationUsage struct {
	ProjectCount  int       `json:"projectCount" bson:"project_count"`
	DatabaseCount int       `json:"databaseCount" bson:"database_count"`
	StorageUsedGB int64     `json:"storageUsedGB" bson:"storage_used_gb"`
	DocumentCount int64     `json:"documentCount" bson:"document_count"`
	ReadsToday    int64     `json:"readsToday" bson:"reads_today"`
	WritesToday   int64     `json:"writesToday" bson:"writes_today"`
	DeletesToday  int64     `json:"deletesToday" bson:"deletes_today"`
	LastUpdated   time.Time `json:"lastUpdated" bson:"last_updated"`
}

// BillingInfo holds billing-related information
type BillingInfo struct {
	BillingAccountID string             `json:"billingAccountId" bson:"billing_account_id"`
	Plan             string             `json:"plan" bson:"plan"` // "free", "pay-as-you-go", "enterprise"
	Currency         string             `json:"currency" bson:"currency"`
	BillingCycle     string             `json:"billingCycle" bson:"billing_cycle"` // "monthly", "annual"
	CustomRates      map[string]float64 `json:"customRates,omitempty" bson:"custom_rates,omitempty"`
}

// IAMPolicy defines access control for the organization
type IAMPolicy struct {
	Bindings []IAMBinding `json:"bindings" bson:"bindings"`
	Version  int          `json:"version" bson:"version"`
}

// IAMBinding represents a role binding
type IAMBinding struct {
	Role    string   `json:"role" bson:"role"`       // e.g., "roles/owner", "roles/editor"
	Members []string `json:"members" bson:"members"` // e.g., "user:admin@company.com"
}

// GetResourceName returns the full resource name for this organization
// Following Firestore naming convention
func (o *Organization) GetResourceName() string {
	return "organizations/" + o.OrganizationID
}

// IsActive returns true if the organization is in active state
func (o *Organization) IsActive() bool {
	return o.State == OrganizationStateActive
}

// CanCreateProject checks if the organization can create more projects
func (o *Organization) CanCreateProject() bool {
	if o.Quotas == nil {
		return true // No quotas set
	}
	return o.ProjectCount < o.Quotas.MaxProjects
}

// HasAdmin checks if the given email is an admin of this organization
func (o *Organization) HasAdmin(email string) bool {
	for _, admin := range o.AdminEmails {
		if admin == email {
			return true
		}
	}
	return o.BillingEmail == email
}

// ValidateOrganizationID validates a Firestore organization ID format
func ValidateOrganizationID(organizationID string) error {
	if organizationID == "" {
		return ErrInvalidOrganizationID
	}

	if len(organizationID) < 3 || len(organizationID) > 30 {
		return ErrInvalidOrganizationID
	}

	// Must start with a letter
	if !((organizationID[0] >= 'a' && organizationID[0] <= 'z') ||
		(organizationID[0] >= 'A' && organizationID[0] <= 'Z')) {
		return ErrInvalidOrganizationID
	}

	// Can contain letters, numbers, hyphens
	for _, char := range organizationID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return ErrInvalidOrganizationID
		}
	}

	return nil
}

// NewOrganization creates a new organization with default values
func NewOrganization(organizationID, displayName, billingEmail string) (*Organization, error) {
	if err := ValidateOrganizationID(organizationID); err != nil {
		return nil, err
	}

	if displayName == "" {
		return nil, ErrInvalidDisplayName
	}

	if billingEmail == "" {
		return nil, ErrInvalidBillingEmail
	}

	now := time.Now()

	return &Organization{
		OrganizationID:  organizationID,
		DisplayName:     displayName,
		BillingEmail:    billingEmail,
		AdminEmails:     []string{billingEmail},
		DefaultLocation: "us-central1",
		State:           OrganizationStatePendingSetup,
		CreatedAt:       now,
		UpdatedAt:       now,
		ProjectCount:    0,
		Quotas: &OrganizationQuotas{
			MaxProjects:      10, // Default free tier
			MaxDatabases:     50,
			MaxStorageGB:     1,     // 1GB free
			MaxDocuments:     20000, // 20K documents free
			MaxReadsPerDay:   50000,
			MaxWritesPerDay:  20000,
			MaxDeletesPerDay: 20000,
		},
		Usage: &OrganizationUsage{
			LastUpdated: now,
		},
		Billing: &BillingInfo{
			Plan:         "free",
			Currency:     "USD",
			BillingCycle: "monthly",
		},
	}, nil
}

// Common organization-related errors
var (
	ErrInvalidOrganizationID = errors.New("invalid organization ID")
	ErrInvalidDisplayName    = errors.New("invalid display name")
	ErrInvalidBillingEmail   = errors.New("invalid billing email")
	ErrOrganizationNotFound  = errors.New("organization not found")
	ErrOrganizationExists    = errors.New("organization already exists")
	ErrOrganizationInactive  = errors.New("organization is not active")
	ErrQuotaExceeded         = errors.New("quota exceeded")
)
