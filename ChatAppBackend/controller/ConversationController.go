package controller

import (
	"ChatApp/initializer"
	"ChatApp/models"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// while creating a conversation first we need to check if the user is already registered or not.
func CheckUserAvailability(c *fiber.Ctx) error {
	// Create a struct to hold the request body
	var userReq struct {
		Email string `json:"email"`
	}

	// Parse the request body into userReq
	if err := c.BodyParser(&userReq); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Check for the availability of the user
	var user models.User
	if err := initializer.DB.Where("email = ?", userReq.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
	}

	// User found, return success
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "User found",
		"userId":  user.ID,
	})
}

// create new Conversation
func CreateConversation(c *fiber.Ctx) error {
	// Parse request body into createConversation struct
	var createConversation models.CreateConversation
	
	if err := c.BodyParser(&createConversation); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to read body"})
	}

	// Fetch conversation IDs involving both participants
	var conversationIDs []uint
	if err := initializer.DB.Table("conversation_users").
		Select("conversation_id").
		Where("user_id IN (?)", createConversation.Participants).
		Group("conversation_id").
		Having("COUNT(DISTINCT user_id) = ?", len(createConversation.Participants)).
		Pluck("conversation_id", &conversationIDs).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch conversations"})
	}

	// Fetch conversation IDs from groups table
	var groupConversationIDs []uint
	if err := initializer.DB.Table("groups").Pluck("conversation_id", &groupConversationIDs).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch group conversations"})
	}

	// Remove group conversation IDs from conversationIDs
	filteredConversationIDs := conversationIDs[:0]
	for _, convID := range conversationIDs {
		if !contains(groupConversationIDs, convID) {
			filteredConversationIDs = append(filteredConversationIDs, convID)
		}
	}

	// If no conversation IDs left, create a new conversation
	if len(filteredConversationIDs) == 0 {
		// Fetch participants based on participant IDs provided
		var participants []models.User
		if err := initializer.DB.Where("id IN (?)", createConversation.Participants).Find(&participants).Error; err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid participants"})
		}

		// Create the conversation object
		conversation := models.Conversation{
			Participants: participants,
			Messages:     make([]models.Message, 0), // Initialize messages slice with empty slice
		}

		// Start a transaction
		tx := initializer.DB.Begin()
		if tx.Error != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": tx.Error.Error()})
		}

		// Save the conversation object within the transaction
		if err := tx.Create(&conversation).Error; err != nil {
			// Rollback the transaction if there's an error
			tx.Rollback()
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create conversation"})
		}

		// Commit the transaction if everything is successful
		tx.Commit()

		// Return the newly created conversation
		return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Conversation created", "conversation": conversation})
	}

	// If there are remaining conversation IDs, it means there's an existing conversation
	return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Conversation already exists"})
}

func GetAllConversations(c *fiber.Ctx) error {

	user, ok := c.Locals("user").(models.User)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve user from context"})
	}
	userID := user.ID

	// Get all conversations where the user participates
	var conversations []models.Conversation
	if err := initializer.DB.Preload("Participants").Preload("Messages.Sender").Joins("JOIN conversation_users ON conversations.id = conversation_users.conversation_id").
		Where("conversation_users.user_id = ?", userID).
		Find(&conversations).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch conversations"})
	}

	// Return the conversations
	return c.Status(http.StatusOK).JSON(fiber.Map{"conversations": conversations})
}

func LeaveConversation(c *fiber.Ctx) error {
	type ReqBody struct {
		UserID         uint `json:"user_id"`
		ConversationID uint `json:"conversation_id"`
	}

	var reqBody ReqBody
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Delete the user_id and conversation_id from the conversation_users table
	if err := initializer.DB.Table("conversation_users").Where("conversation_id = ? AND user_id = ?", reqBody.ConversationID, reqBody.UserID).Delete(nil).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to leave conversation"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Left conversation successfully"})
}

func GetAllPersonalChats(c *fiber.Ctx) error {
	// Get user from context
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve user from context"})
	}
	userID := user.ID

	// Get all conversations where the user participates
	var allConversations []models.Conversation
	if err := initializer.DB.Preload("Participants").Preload("Messages.Sender").
		Joins("JOIN conversation_users ON conversations.id = conversation_users.conversation_id").
		Where("conversation_users.user_id = ?", userID).
		Find(&allConversations).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch conversations"})
	}

	// Get the IDs of group conversations
	var groups []models.Group
	if err := initializer.DB.Preload("GroupConversation").
		Preload("GroupConversation.Participants").
		Where("admin_id = ? OR conversation_id IN (SELECT conversation_id FROM conversation_users WHERE user_id = ?)", userID, userID).
		Find(&groups).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch groups"})
	}

	// get all groups conversation id from group object
	var groupConversationIDs []uint
	for _, group := range groups {
		groupConversationIDs = append(groupConversationIDs, group.GroupConversation.ID)
	}

	// Filter out personal conversations
	var personalChats []models.Conversation
	for _, conversation := range allConversations {
		if !contains(groupConversationIDs, conversation.ID) && len(conversation.Participants) == 2 {
			personalChats = append(personalChats, conversation)
		}
	}

	//if personalChats are empty then return empty []
	if len(personalChats) == 0 {
		return c.Status(http.StatusOK).JSON(fiber.Map{"personalChats": []models.Conversation{}})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"personalChats": personalChats})
}

// contains checks if a given element exists in a slice
func contains(slice []uint, element uint) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
