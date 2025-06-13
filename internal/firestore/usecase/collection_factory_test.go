package usecase

import (
	"errors"
	"testing"

	"firestore-clone/internal/firestore/domain/repository"
)

type mockNamingStrategy struct {
	returnError bool
}

func (m *mockNamingStrategy) CollectionName(projectID, databaseID, collectionPath string) string {
	if m.returnError {
		return ""
	}
	return projectID + ":" + databaseID + ":" + collectionPath
}

type mockFactory struct {
	returnError bool
}

func (m *mockFactory) GetCollection(projectID, databaseID, collectionPath string) (repository.CollectionReference, error) {
	if m.returnError {
		return nil, errors.New("error")
	}
	return &DefaultCollectionReference{name: projectID + ":" + databaseID + ":" + collectionPath}, nil
}

func TestDefaultCollectionFactory_GetCollection(t *testing.T) {
	strategy := &mockNamingStrategy{}
	factory := NewDefaultCollectionFactory(strategy)
	ref, err := factory.GetCollection("p1", "db1", "col1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Name() != "p1:db1:col1" {
		t.Errorf("expected name p1:db1:col1, got %s", ref.Name())
	}
}

func TestInMemoryCollectionManager_Cache(t *testing.T) {
	factory := &mockFactory{}
	manager := NewInMemoryCollectionManager(factory)
	ref1, err := manager.GetOrCreateCollection("p1", "db1", "col1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ref2, err := manager.GetOrCreateCollection("p1", "db1", "col1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref1 != ref2 {
		t.Errorf("expected cached reference, got different instances")
	}
}

func TestInMemoryCollectionManager_Error(t *testing.T) {
	factory := &mockFactory{returnError: true}
	manager := NewInMemoryCollectionManager(factory)
	_, err := manager.GetOrCreateCollection("p1", "db1", "col1")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
