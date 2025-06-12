package http

import (
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Project handlers implementation following single responsibility principle
func (h *HTTPHandler) CreateProject(c *fiber.Ctx) error {
	h.Log.Debug("Creating project via HTTP")

	var req usecase.CreateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	project, err := h.FirestoreUC.CreateProject(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to create project", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_project_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Project created successfully", "projectID", project.ProjectID)
	return c.Status(fiber.StatusCreated).JSON(project)
}

func (h *HTTPHandler) GetProject(c *fiber.Ctx) error {
	h.Log.Debug("Getting project via HTTP", "projectID", c.Params("projectID"))

	req := usecase.GetProjectRequest{
		ProjectID: c.Params("projectID"),
	}

	project, err := h.FirestoreUC.GetProject(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to get project", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "project_not_found",
			"message": err.Error(),
		})
	}

	return c.JSON(project)
}

func (h *HTTPHandler) UpdateProject(c *fiber.Ctx) error {
	h.Log.Debug("Updating project via HTTP", "projectID", c.Params("projectID"))

	var req usecase.UpdateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Ensure projectID from path is used
	if req.Project != nil {
		req.Project.ProjectID = c.Params("projectID")
	}

	project, err := h.FirestoreUC.UpdateProject(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to update project", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_project_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Project updated successfully", "projectID", project.ProjectID)
	return c.JSON(project)
}

func (h *HTTPHandler) DeleteProject(c *fiber.Ctx) error {
	h.Log.Debug("Deleting project via HTTP", "projectID", c.Params("projectID"))

	req := usecase.DeleteProjectRequest{
		ProjectID: c.Params("projectID"),
	}

	err := h.FirestoreUC.DeleteProject(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to delete project", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_project_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Project deleted successfully", "projectID", req.ProjectID)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandler) ListProjects(c *fiber.Ctx) error {
	h.Log.Debug("Listing projects via HTTP")
	// Extract organizationId from URL path parameter
	organizationID := c.Params("organizationId")
	trimmedOrgID := strings.TrimSpace(organizationID)

	if trimmedOrgID == "" || model.ValidateOrganizationID(trimmedOrgID) != nil {
		h.Log.Error("Missing or invalid organization ID in URL path")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_organization_id",
			"message": "Organization ID is required in the URL path and must be valid",
		})
	}

	req := usecase.ListProjectsRequest{
		OrganizationID: trimmedOrgID,
		OwnerEmail:     c.Query("ownerEmail"), // Optional filter by owner
	}
	projects, err := h.FirestoreUC.ListProjects(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to list projects", "error", err, "organizationID", trimmedOrgID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_projects_failed",
			"message": err.Error(),
		})
	}

	h.Log.Debug("Projects listed successfully", "count", len(projects), "organizationID", trimmedOrgID)
	return c.JSON(fiber.Map{
		"projects": projects,
		"count":    len(projects),
	})
}
