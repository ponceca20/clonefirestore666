package usecase

import (
	"firestore-clone/internal/firestore/domain/repository"
	"sync"
)

type DefaultCollectionFactory struct {
	strategy repository.CollectionNamingStrategy
}

func NewDefaultCollectionFactory(strategy repository.CollectionNamingStrategy) *DefaultCollectionFactory {
	return &DefaultCollectionFactory{strategy: strategy}
}

func (f *DefaultCollectionFactory) GetCollection(projectID, databaseID, collectionPath string) (repository.CollectionReference, error) {
	name := f.strategy.CollectionName(projectID, databaseID, collectionPath)
	return &DefaultCollectionReference{name: name}, nil
}

type DefaultCollectionReference struct {
	name string
}

func (c *DefaultCollectionReference) Name() string {
	return c.name
}

// CollectionManager with simple in-memory cache

type InMemoryCollectionManager struct {
	factory repository.CollectionFactory
	cache   map[string]repository.CollectionReference
	mu      sync.Mutex
}

func NewInMemoryCollectionManager(factory repository.CollectionFactory) *InMemoryCollectionManager {
	return &InMemoryCollectionManager{
		factory: factory,
		cache:   make(map[string]repository.CollectionReference),
	}
}

func (m *InMemoryCollectionManager) GetOrCreateCollection(projectID, databaseID, collectionPath string) (repository.CollectionReference, error) {
	key := projectID + "." + databaseID + "." + collectionPath
	m.mu.Lock()
	defer m.mu.Unlock()
	if ref, ok := m.cache[key]; ok {
		return ref, nil
	}
	ref, err := m.factory.GetCollection(projectID, databaseID, collectionPath)
	if err != nil {
		return nil, err
	}
	m.cache[key] = ref
	return ref, nil
}
