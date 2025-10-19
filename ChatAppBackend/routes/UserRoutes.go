package routes

import (
	"ChatApp/controller"
	"ChatApp/middleware"

	"github.com/gofiber/fiber/v2"
)

func UserRoutes(app *fiber.App) {

	user := app.Group("/user")
	{
		//user controller
		user.Post("/otp", controller.OTPSendToEmail)
		user.Post("/otp-verify", controller.OTPVerify)
		user.Post("/signup", controller.SignUp)
		user.Post("/login", controller.Login)
		user.Post("/change-password", controller.ChangePassword)
		user.Post("/editProfile", middleware.RequireAuth, controller.UpdateUser)
		user.Get("/validate", middleware.RequireAuth, controller.Validate)

		//Conversation Controller
		user.Post("/createConversation", middleware.RequireAuth, controller.CreateConversation)
		user.Post("/avalibility", middleware.RequireAuth, controller.CheckUserAvailability)
		user.Get("/allConversations", middleware.RequireAuth, controller.GetAllConversations)
		user.Get("/personalChats", middleware.RequireAuth, controller.GetAllPersonalChats)
		user.Post("/leaveConversation", middleware.RequireAuth, controller.LeaveConversation)

		//Group Controller
		user.Post("/createGroup", middleware.RequireAuth, controller.CreateGroup)
		user.Post("/addToGroup", middleware.RequireAuth, controller.AddToGroup)
		user.Post("/leaveGroup", middleware.RequireAuth, controller.LeaveGroup)
		user.Get("/groupChats", middleware.RequireAuth, controller.GetAllGroups)
		user.Post("/removeMember", middleware.RequireAuth, controller.RemoveGroupMember)
		user.Post("/deleteGroup", middleware.RequireAuth, controller.DeleteGroup)
		user.Post("/isGroupAndConversation", middleware.RequireAuth, controller.IsGroupAndConversation)

	}
}
