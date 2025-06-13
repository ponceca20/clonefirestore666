package mongodb

import (
	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MockSingleResult permite simular resultados de FindOne
// Si se busca una organización con OrganizationID == "non-existent-org", retorna mongo.ErrNoDocuments

type MockSingleResult struct {
	Filter interface{}
}

func (m MockSingleResult) Decode(v interface{}) error {
	if m.Filter != nil {
		switch filter := m.Filter.(type) {
		case map[string]interface{}:
			if id, ok := filter["organization_id"]; ok && id == "non-existent-org" {
				return mongo.ErrNoDocuments
			}
		case bson.M:
			if id, ok := filter["organization_id"]; ok && id == "non-existent-org" {
				return mongo.ErrNoDocuments
			}
		}
	}
	if idx, ok := v.(*model.Index); ok {
		idx.Name = "test_index"
		return nil
	}
	return nil
}
