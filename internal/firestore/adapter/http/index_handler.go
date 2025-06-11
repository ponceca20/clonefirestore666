package http

import (
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
)

// Index handlers implementation following single responsibility principle
func (h *HTTPHandler) CreateIndex(c *fiber.Ctx) error {
	h.Log.Debug("Creating index via HTTP", "collection", c.Params("collectionID"))

	var req usecase.CreateIndexRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")
	req.DatabaseID = c.Params("databaseID")

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Set collection from path if not provided in body
	if req.Index.Collection == "" {
		req.Index.Collection = c.Params("collectionID")
	}

	index, err := h.FirestoreUC.CreateIndex(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to create index", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_index_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Index created successfully", "indexName", index.Name)
	return c.Status(fiber.StatusCreated).JSON(index)
}

func (h *HTTPHandler) ListIndexes(c *fiber.Ctx) error {
	h.Log.Debug("Listing indexes via HTTP", "collection", c.Params("collectionID"))

	req := usecase.ListIndexesRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
	}

	indexes, err := h.FirestoreUC.ListIndexes(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to list indexes", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_indexes_failed",
			"message": err.Error(),
		})
	}

	h.Log.Debug("Indexes listed successfully", "count", len(indexes))
	return c.JSON(fiber.Map{
		"indexes": indexes,
		"count":   len(indexes),
	})
}

func (h *HTTPHandler) DeleteIndex(c *fiber.Ctx) error {
	h.Log.Debug("Deleting index via HTTP",
		"collection", c.Params("collectionID"),
		"index", c.Params("indexID"))

	req := usecase.DeleteIndexRequest{
		ProjectID:  c.Params("projectID"),
		DatabaseID: c.Params("databaseID"),
		IndexName:  c.Params("indexID"),
	}

	err := h.FirestoreUC.DeleteIndex(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to delete index", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_index_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Index deleted successfully", "indexName", req.IndexName)
	return c.SendStatus(fiber.StatusNoContent)
}
