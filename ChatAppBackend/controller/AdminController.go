package controller

import (
	"ChatApp/initializer"
	"ChatApp/models"
	"github.com/gofiber/fiber/v2"
	"net/http"
)

// Admin can remove any user
func RemoveUser(c *fiber.Ctx) error {
	// Create a struct to hold the request body
	type ReqBody struct {
		AdminID uint `json:"admin_id"`
		UserID  uint `json:"user_id"`
	}
	//parse request body
	var reqBody ReqBody
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to parse request body",
		})
	}
	// Find admin user from admin ID in the user table
	var adminUser models.User
	if err := initializer.DB.First(&adminUser, reqBody.AdminID).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Admin user not found",
		})
	}
	// Check if the admin is indeed an admin
	if !adminUser.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "User is not an admin",
		})
	}
	// Check for the user to delete from the user ID in the user table
	var userToDelete models.User
	if err := initializer.DB.First(&userToDelete, reqBody.UserID).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "User to delete not found",
		})
	}
	// Delete the user from the user table
	if err := initializer.DB.Delete(&userToDelete).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}
	//return updated user list who are not admins
	var users []models.User
	if err := initializer.DB.Find(&users).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get users",
		})
	}
	// remove all the users from users[] who are admin
	for i := 0; i < len(users); i++ {
		if users[i].IsAdmin {
			users = append(users[:i], users[i+1:]...)
			i--
		}
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "User deleted successfully",
		"users":   users,
	})
}

// get all users
func GetAllUsers(c *fiber.Ctx) error {
	//check the user is admin or not admin
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve user from context"})
	}
	if !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "User is not an admin",
		})
	}
	var users []models.User
	if err := initializer.DB.Find(&users).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get users",
		})
	}
	// remove all the users from users[] who are admin
	for i := 0; i < len(users); i++ {
		if users[i].IsAdmin {
			users = append(users[:i], users[i+1:]...)
			i--
		}
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Users retrieved successfully",
		"users":   users,
	})

}
