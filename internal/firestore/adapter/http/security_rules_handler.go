package http

import (
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
)

type SecurityRulesHandler struct {
	UC usecase.SecurityRulesCRUDUsecase
}

func NewSecurityRulesHandler(uc usecase.SecurityRulesCRUDUsecase) *SecurityRulesHandler {
	return &SecurityRulesHandler{UC: uc}
}

func (h *SecurityRulesHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/v1/projects/:projectID/databases/:databaseID/securityRules", h.GetRules)
	router.Put("/v1/projects/:projectID/databases/:databaseID/securityRules", h.PutRules)
	router.Patch("/v1/projects/:projectID/databases/:databaseID/securityRules", h.PatchRules)
	router.Delete("/v1/projects/:projectID/databases/:databaseID/securityRules", h.DeleteRules)
	router.Post("/v1/projects/:projectID/databases/:databaseID/securityRules:validate", h.ValidateRules)
}

func (h *SecurityRulesHandler) GetRules(c *fiber.Ctx) error {
	rules, err := h.UC.GetRules(c.UserContext(), c.Params("projectID"), c.Params("databaseID"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendString(rules)
}

func (h *SecurityRulesHandler) PutRules(c *fiber.Ctx) error {
	var body struct {
		Rules string `json:"rules"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid body"})
	}
	err := h.UC.PutRules(c.UserContext(), c.Params("projectID"), c.Params("databaseID"), body.Rules, "admin")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *SecurityRulesHandler) PatchRules(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"error": "PATCH not implemented"})
}

func (h *SecurityRulesHandler) DeleteRules(c *fiber.Ctx) error {
	err := h.UC.DeleteRules(c.UserContext(), c.Params("projectID"), c.Params("databaseID"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *SecurityRulesHandler) ValidateRules(c *fiber.Ctx) error {
	var body struct {
		Rules string `json:"rules"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid body"})
	}
	err := h.UC.ValidateRules(c.UserContext(), body.Rules)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}
