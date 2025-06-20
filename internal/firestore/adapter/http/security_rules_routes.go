package http

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterSecurityRulesRoutes integra el handler de reglas de seguridad al router principal
func RegisterSecurityRulesRoutes(app *fiber.App, handler *SecurityRulesHandler) {
	handler.RegisterRoutes(app)
}
