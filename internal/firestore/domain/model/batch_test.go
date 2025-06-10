package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBatch_Compile(t *testing.T) {
	// Placeholder: Add real batch model tests here
}

func TestBatchWriteOperation_ModelFields(t *testing.T) {
	op := BatchWriteOperation{
		Type:       BatchOperationTypeCreate,
		DocumentID: "doc1",
		Path:       "projects/p1/databases/d1/documents/c1/doc1",
		Data:       map[string]any{"field": "value"},
		Mask:       []string{"field"},
	}
	assert.Equal(t, BatchOperationTypeCreate, op.Type)
	assert.Equal(t, "doc1", op.DocumentID)
	assert.Equal(t, "value", op.Data["field"])
	assert.Contains(t, op.Mask, "field")
}

func TestBatchWriteRequest_ModelFields(t *testing.T) {
	req := BatchWriteRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
		Operations: []BatchWriteOperation{{Type: BatchOperationTypeUpdate}},
		Labels:     map[string]string{"env": "test"},
	}
	assert.Equal(t, "p1", req.ProjectID)
	assert.Equal(t, BatchOperationTypeUpdate, req.Operations[0].Type)
	assert.Equal(t, "test", req.Labels["env"])
}

func TestBatchWriteResponse_ModelFields(t *testing.T) {
	resp := BatchWriteResponse{
		WriteResults: []WriteResult{{UpdateTime: time.Now()}},
		Status:       []Status{{Code: 0, Message: "ok"}},
	}
	assert.Len(t, resp.WriteResults, 1)
	assert.Equal(t, int32(0), resp.Status[0].Code)
}
