package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProject(t *testing.T) {
	uc := newTestFirestoreUsecase()
	proj, err := uc.CreateProject(context.Background(), CreateProjectRequest{
		Project: &model.Project{ProjectID: "p1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "p1", proj.ProjectID)
}

func TestGetProject(t *testing.T) {
	uc := newTestFirestoreUsecase()
	proj, err := uc.GetProject(context.Background(), GetProjectRequest{
		ProjectID: "p1",
	})
	require.NoError(t, err)
	assert.Equal(t, "p1", proj.ProjectID)
}

func TestUpdateProject(t *testing.T) {
	uc := newTestFirestoreUsecase()
	proj, err := uc.UpdateProject(context.Background(), UpdateProjectRequest{
		Project: &model.Project{ProjectID: "p1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "p1", proj.ProjectID)
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
		OwnerEmail: "test@example.com",
	})
	require.NoError(t, err)
	assert.Len(t, projs, 1)
	assert.Equal(t, "p1", projs[0].ProjectID)
}
