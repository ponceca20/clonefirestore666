package http

import (
	"context"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/mock"
)

// MockOrganizationRepo is a mock implementation of OrganizationRepo interface
type MockOrganizationRepo struct {
	mock.Mock
}

func (m *MockOrganizationRepo) CreateOrganization(ctx context.Context, org *model.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationRepo) GetOrganization(ctx context.Context, id string) (*model.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Organization), args.Error(1)
}

func (m *MockOrganizationRepo) ListOrganizations(ctx context.Context, pageSize, offset int) ([]*model.Organization, error) {
	args := m.Called(ctx, pageSize, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Organization), args.Error(1)
}

func (m *MockOrganizationRepo) ListOrganizationsByAdmin(ctx context.Context, adminEmail string) ([]*model.Organization, error) {
	args := m.Called(ctx, adminEmail)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Organization), args.Error(1)
}

func (m *MockOrganizationRepo) UpdateOrganization(ctx context.Context, org *model.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationRepo) DeleteOrganization(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
