package routes

import (
	"ChatApp/controller"
	"ChatApp/middleware"
	"github.com/gofiber/fiber/v2"
)

func AdminRoutes(app *fiber.App) {

	admin := app.Group("/admin")
	{
		admin.Get("/allUsers", middleware.RequireAuth, controller.GetAllUsers)
		admin.Post("/removeUser", middleware.RequireAuth, controller.RemoveUser)
	}
}
