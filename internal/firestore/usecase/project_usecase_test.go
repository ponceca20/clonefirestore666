package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFirestoreUsecase() FirestoreUsecaseInterface {
	return NewFirestoreUsecase(
		NewMockFirestoreRepo(), // Mock aislado por test
		nil,
		nil,
		&MockLogger{},
	)
}

func TestCreateProject(t *testing.T) {
	uc := newTestFirestoreUsecase()
	uniqueProjectID := "p_createproject_" + t.Name()
	uniqueOrgID := "org_createproject_" + t.Name()
	proj, err := uc.CreateProject(context.Background(), CreateProjectRequest{
		Project: &model.Project{ProjectID: uniqueProjectID, OrganizationID: uniqueOrgID},
	})
	require.NoError(t, err)
	assert.Equal(t, uniqueProjectID, proj.ProjectID)
}

func TestGetProject(t *testing.T) {
	uc := newTestFirestoreUsecase()
	projID := "p1"
	orgID := "org1"
	// Insertar el proyecto antes de intentar obtenerlo
	_, err := uc.CreateProject(context.Background(), CreateProjectRequest{
		Project: &model.Project{ProjectID: projID, OrganizationID: orgID},
	})
	require.NoError(t, err)
	proj, err := uc.GetProject(context.Background(), GetProjectRequest{
		ProjectID: projID,
	})
	require.NoError(t, err)
	assert.Equal(t, projID, proj.ProjectID)
}

func TestUpdateProject(t *testing.T) {
	uc := newTestFirestoreUsecase()
	projID := "p1"
	orgID := "org1"
	// Insertar el proyecto antes de intentar actualizarlo
	_, err := uc.CreateProject(context.Background(), CreateProjectRequest{
		Project: &model.Project{ProjectID: projID, OrganizationID: orgID},
	})
	require.NoError(t, err)
	proj, err := uc.UpdateProject(context.Background(), UpdateProjectRequest{
		Project: &model.Project{ProjectID: projID},
	})
	require.NoError(t, err)
	assert.Equal(t, projID, proj.ProjectID)
}

func TestDeleteProject(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.DeleteProject(context.Background(), DeleteProjectRequest{
		ProjectID: "p1",
	})
	assert.NoError(t, err)
}

func TestListProjects(t *testing.T) {
	uc := newTestFirestoreUsecase()
	projs, err := uc.ListProjects(context.Background(), ListProjectsRequest{
		OrganizationID: "test-org",
		OwnerEmail:     "test@example.com",
	})
	require.NoError(t, err)
	assert.Len(t, projs, 1)
	assert.Equal(t, "p1", projs[0].ProjectID)
}
