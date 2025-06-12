# Projects Collection Fixes

## Summary of Changes

The `proyects_colection.json` Postman collection has been updated to correctly align with the actual API routes and code implementation.

## Issues Fixed

### 1. Route Path Correction
- **Before**: `/api/v1/organizations/{{organizationId}}/projects`
- **After**: `/organizations/{{organizationId}}/projects`
- **Reason**: The actual routes are registered without the `/api/v1` prefix

### 2. Parameter Naming Consistency
- **Before**: `{{projectId}}` (camelCase with lowercase 'd')
- **After**: `{{projectID}}` (camelCase with uppercase 'D')
- **Reason**: The code uses `:projectID` parameter and `projectID` field name

### 3. Request Body Structure Alignment
- **Before**: Used deprecated fields like `description`, `defaultLocation`, `labels`
- **After**: Uses actual model fields: `projectID`, `displayName`, `organizationId`, `locationId`, `ownerEmail`, `state`
- **Reason**: Aligned with the actual `Project` model structure in `internal/firestore/domain/model/project.go`

### 4. Route Conflict Resolution
- **Issue**: There were conflicting routes between `OrganizationHandler.ListOrganizationProjects` and `HTTPHandler.ListProjects`
- **Fix**: Removed the placeholder `ListOrganizationProjects` method and let the main project handlers handle all project operations
- **Result**: Clean route hierarchy with no conflicts

### 5. Enhanced Test Scripts
- Added comprehensive test scripts for each endpoint
- Added validation for response structure
- Added error case testing
- Added automatic environment variable setting

## Updated Routes

All project routes now follow this pattern:
```
Base URL: {{baseUrl}}/organizations/{{organizationId}}/projects
```

### Available Endpoints:
1. **POST** `/organizations/{{organizationId}}/projects` - Create Project
2. **GET** `/organizations/{{organizationId}}/projects` - List Projects
3. **GET** `/organizations/{{organizationId}}/projects/{{projectID}}` - Get Project
4. **PUT** `/organizations/{{organizationId}}/projects/{{projectID}}` - Update Project
5. **DELETE** `/organizations/{{organizationId}}/projects/{{projectID}}` - Delete Project

### Error Testing Endpoints:
6. **POST** `/organizations/{{organizationId}}/projects` - Create Project with Invalid Data (400 error)
7. **GET** `/organizations/{{organizationId}}/projects/nonexistent-project-id` - Get Non-existent Project (404 error)

## Environment Variables

The collection now uses these environment variables:
- `baseUrl`: Base API URL (default: `http://localhost:3030`)
- `organizationId`: Organization ID for testing
- `projectID`: Current project ID (set automatically after project creation)
- `newProjectID`: Generated project ID for creating new projects
- `ownerEmail`: Email for filtering projects by owner
- `authToken`: Authentication token for API access

## Request/Response Examples

### Create Project Request Body:
```json
{
  "project": {
    "projectID": "{{newProjectID}}",
    "displayName": "My Awesome Project",
    "organizationId": "{{organizationId}}",
    "locationId": "us-central1",
    "ownerEmail": "admin@example.com",
    "state": "ACTIVE"
  }
}
```

### Expected Response Structure:
```json
{
  "id": "...",
  "projectID": "project-123",
  "displayName": "My Awesome Project",
  "organizationId": "org-456",
  "locationId": "us-central1",
  "ownerEmail": "admin@example.com",
  "state": "ACTIVE",
  "createdAt": "2025-06-12T...",
  "updatedAt": "2025-06-12T...",
  "collaborators": [],
  "resources": {...}
}
```

## Code Changes Made

### 1. Fixed Organization Handler Registration
- Updated `registerOrganizationRoutes` in `http_handler_main.go` to actually call `OrganizationHandler.RegisterRoutes()`

### 2. Removed Route Conflicts
- Removed `ListOrganizationProjects` method from `OrganizationHandler`
- Updated `OrganizationHandler.RegisterRoutes()` to not register conflicting project routes
- Removed obsolete tests for the removed method

### 3. Clean Architecture
- Organization Handler: Manages organization CRUD operations
- HTTP Handler: Manages project CRUD operations under organization hierarchy
- Clear separation of concerns with no overlapping routes

## Testing

All tests pass:
- ✅ Project handler tests
- ✅ Organization handler tests  
- ✅ API integration tests

## Usage Instructions

1. Import the updated `proyects_colection.json` into Postman
2. Set up your environment variables (especially `baseUrl`, `organizationId`, and `authToken`)
3. Run the requests in order:
   - Create Project (sets `projectID` automatically)
   - List Projects 
   - Get Project
   - Update Project
   - Delete Project
4. Use the error testing endpoints to verify error handling

## Architecture Notes

The API follows the Firestore hierarchy:
```
Organizations → Projects → Databases → Documents
```

All project operations are scoped to an organization, ensuring proper tenant isolation and access control.
