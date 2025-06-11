package http

import (
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
)

// Collection handlers implementation following single responsibility principle
func (h *HTTPHandler) CreateCollection(c *fiber.Ctx) error {
	h.Log.Debug("Creating collection via HTTP", "collection", c.Params("collectionID"))

	req := usecase.CreateCollectionRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
	}

	// If collectionID is not in path, try to get it from body
	if req.CollectionID == "" {
		var bodyReq struct {
			CollectionID string `json:"collectionId"`
		}
		if err := c.BodyParser(&bodyReq); err == nil && bodyReq.CollectionID != "" {
			req.CollectionID = bodyReq.CollectionID
		}
	}

	if req.CollectionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_collection_id",
			"message": "Collection ID is required",
		})
	}

	collection, err := h.FirestoreUC.CreateCollection(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to create collection", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_collection_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Collection created successfully", "collectionID", collection.CollectionID)
	return c.Status(fiber.StatusCreated).JSON(collection)
}

func (h *HTTPHandler) GetCollection(c *fiber.Ctx) error {
	h.Log.Debug("Getting collection via HTTP", "collection", c.Params("collectionID"))

	req := usecase.GetCollectionRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
	}

	collection, err := h.FirestoreUC.GetCollection(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to get collection", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "collection_not_found",
			"message": err.Error(),
		})
	}

	return c.JSON(collection)
}

func (h *HTTPHandler) UpdateCollection(c *fiber.Ctx) error {
	h.Log.Debug("Updating collection via HTTP", "collection", c.Params("collectionID"))

	var collection struct {
		DisplayName string            `json:"displayName,omitempty"`
		Metadata    map[string]string `json:"metadata,omitempty"`
	}

	if err := c.BodyParser(&collection); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	req := usecase.UpdateCollectionRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		Collection: &model.Collection{
			CollectionID: c.Params("collectionID"),
			ProjectID:    c.Params("projectID"),
			DatabaseID:   c.Params("databaseID"),
			DisplayName:  collection.DisplayName,
		},
	}

	err := h.FirestoreUC.UpdateCollection(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to update collection", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_collection_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Collection updated successfully", "collectionID", req.CollectionID)
	return c.SendStatus(fiber.StatusOK)
}

func (h *HTTPHandler) ListCollections(c *fiber.Ctx) error {
	h.Log.Debug("Listing collections via HTTP")

	req := usecase.ListCollectionsRequest{
		ProjectID:  c.Params("projectID"),
		DatabaseID: c.Params("databaseID"),
	}

	// Parse optional query parameters
	if pageSize := c.QueryInt("pageSize"); pageSize > 0 {
		req.PageSize = int32(pageSize)
	}
	req.PageToken = c.Query("pageToken")

	collections, err := h.FirestoreUC.ListCollections(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to list collections", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_collections_failed",
			"message": err.Error(),
		})
	}

	h.Log.Debug("Collections listed successfully", "count", len(collections))
	return c.JSON(fiber.Map{
		"collections": collections,
		"count":       len(collections),
	})
}

func (h *HTTPHandler) DeleteCollection(c *fiber.Ctx) error {
	h.Log.Debug("Deleting collection via HTTP", "collection", c.Params("collectionID"))

	req := usecase.DeleteCollectionRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
	}

	err := h.FirestoreUC.DeleteCollection(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to delete collection", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_collection_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Collection deleted successfully", "collectionID", req.CollectionID)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandler) ListSubcollections(c *fiber.Ctx) error {
	h.Log.Debug("Listing subcollections via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	req := usecase.ListSubcollectionsRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
	}

	subcollections, err := h.FirestoreUC.ListSubcollections(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to list subcollections", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_subcollections_failed",
			"message": err.Error(),
		})
	}

	h.Log.Debug("Subcollections listed successfully", "count", len(subcollections))
	return c.JSON(fiber.Map{
		"subcollections": subcollections,
		"count":          len(subcollections),
	})
}
