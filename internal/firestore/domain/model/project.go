package model

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Project represents a Firestore project in the hierarchy
// Organization → Project → Database → Documents
// Following Firestore's exact hierarchy: organizations/{ORG_ID}/projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/...
type Project struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ProjectID   string             `json:"projectId" bson:"project_id"`
	DisplayName string             `json:"displayName" bson:"display_name"`

	// Organization relationship (NEW - Following Firestore hierarchy)
	OrganizationID string `json:"organizationId" bson:"organization_id"`

	// Project configuration
	LocationID   string            `json:"locationId" bson:"location_id"`
	DefaultAppID string            `json:"defaultAppId,omitempty" bson:"default_app_id,omitempty"`
	Resources    *ProjectResources `json:"resources,omitempty" bson:"resources,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updated_at"`

	// Ownership and access control
	OwnerEmail    string   `json:"ownerEmail" bson:"owner_email"`
	Collaborators []string `json:"collaborators,omitempty" bson:"collaborators,omitempty"`

	// Project state
	State     ProjectState `json:"state" bson:"state"`
	DeletedAt *time.Time   `json:"deletedAt,omitempty" bson:"deleted_at,omitempty"`
}

// ProjectState represents the current state of a project
type ProjectState string

const (
	ProjectStateActive   ProjectState = "ACTIVE"
	ProjectStateDeleted  ProjectState = "DELETE_REQUESTED"
	ProjectStateDisabled ProjectState = "DISABLED"
)

// ProjectResources contains resource configuration for the project
type ProjectResources struct {
	// Resource name in the format: projects/{PROJECT_ID}
	Name string `json:"name" bson:"name"`

	// Firebase project number (for compatibility)
	ProjectNumber string `json:"projectNumber,omitempty" bson:"project_number,omitempty"`

	// Quota and limits
	StorageQuota  int64 `json:"storageQuota" bson:"storage_quota"`
	DocumentQuota int64 `json:"documentQuota" bson:"document_quota"`
	RequestQuota  int64 `json:"requestQuota" bson:"request_quota"`

	// Resource usage tracking
	StorageUsed   int64 `json:"storageUsed" bson:"storage_used"`
	DocumentsUsed int64 `json:"documentsUsed" bson:"documents_used"`
	RequestsToday int64 `json:"requestsToday" bson:"requests_today"`
}

// ValidateProjectID validates a Firestore project ID format
func ValidateProjectID(projectID string) error {
	if len(projectID) == 0 {
		return ErrInvalidProjectID
	}

	// Project ID must be 6-30 characters, lowercase letters, numbers, and hyphens
	// Must start with a letter
	if len(projectID) < 6 || len(projectID) > 30 {
		return ErrInvalidProjectIDLength
	}

	// Check if starts with letter and contains only valid characters
	for i, r := range projectID {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
				return ErrInvalidProjectIDFormat
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
				return ErrInvalidProjectIDFormat
			}
		}
	}

	return nil
}

// GetResourceName returns the full resource name for this project
// Following Firestore hierarchy: organizations/{ORG_ID}/projects/{PROJECT_ID}
func (p *Project) GetResourceName() string {
	if p.OrganizationID != "" {
		return "organizations/" + p.OrganizationID + "/projects/" + p.ProjectID
	}
	return "projects/" + p.ProjectID // Fallback for backward compatibility
}

// GetFullHierarchyPath returns the full path including organization
func (p *Project) GetFullHierarchyPath() string {
	return "organizations/" + p.OrganizationID + "/projects/" + p.ProjectID
}

// IsActive returns true if the project is in active state
func (p *Project) IsActive() bool {
	return p.State == ProjectStateActive
}

// CanAccess checks if the given user email has access to this project
func (p *Project) CanAccess(userEmail string) bool {
	if p.OwnerEmail == userEmail {
		return true
	}

	for _, collaborator := range p.Collaborators {
		if collaborator == userEmail {
			return true
		}
	}

	return false
}

// Common project-related errors
var (
	ErrInvalidProjectID       = errors.New("project ID cannot be empty")
	ErrInvalidProjectIDLength = errors.New("project ID must be 6-30 characters")
	ErrInvalidProjectIDFormat = errors.New("project ID must start with letter and contain only lowercase letters, numbers, and hyphens")
	ErrProjectNotFound        = errors.New("project not found")
	ErrProjectAccessDenied    = errors.New("access denied to project")
	ErrProjectInactive        = errors.New("project is not active")
)
