package controller

import (
	"ChatApp/initializer"
	"ChatApp/models"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Create New Group
func CreateGroup(c *fiber.Ctx) error {

	// take userid , participant and group name from req body
	type GrpReqBody struct {
		UserId       uint   `json:"userId"`
		GroupName    string `json:"groupName"`
		Participants []int  `json:"participants"`
	}

	var createGrp GrpReqBody

	if err := c.BodyParser(&createGrp); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	//find all the participants and  add them to array
	var grpMembers []models.User
	if err := initializer.DB.Where("id IN (?)",
		createGrp.Participants).Find(&grpMembers).Error; err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid participants",
		})
	}

	//create conversation object
	conversation := models.Conversation{
		Participants: grpMembers,
		Messages:     []models.Message{},
	}

	//find Admin
	var admin models.User
	if err := initializer.DB.First(&admin, createGrp.UserId).Error; err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get group admin",
		})
	}

	//create group object
	group := models.Group{
		GroupName:         createGrp.GroupName,
		ConversationID:    conversation.ID,
		GroupConversation: conversation,
		AdminId:           admin.ID,
		Admin:             admin,
	}

	//start a transaction
	tx := initializer.DB.Begin()
	if tx.Error != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": tx.Error.Error(),
		})
	}

	//save the Group Object
	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create group",
		})
	}

	//commit transation if everything is ok
	tx.Commit()
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message":        "Group created successfully",
		"groupCreatedBy": createGrp.UserId,
		"group":          group,
	})
}

// Add new Members to existing group
func AddToGroup(c *fiber.Ctx) error {
	// Parse request body
	type reqBody struct {
		ConversationId uint `json:"conversationId"`
		New_user_id    uint `json:"newUserId"`
	}

	var newMember reqBody
	if err := c.BodyParser(&newMember); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Create a new conversation object
	conversation := models.Conversation{}

	// Retrieve the conversation from the database
	if err := initializer.DB.First(&conversation, newMember.ConversationId).Error; err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to find conversation",
		})
	}

	// Create a new user object
	user := models.User{}
	// Retrieve the user from the database
	if err := initializer.DB.First(&user, newMember.New_user_id).Error; err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to find user",
		})
	}

	// Add the user to the participants of the conversation
	initializer.DB.Model(&conversation).Association("Participants").Append(&user)

	// Preload participants
	if err := initializer.DB.Preload("Participants").First(&conversation, newMember.ConversationId).Error; err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to preload conversation participants",
		})
	}

	// Return success response with participants
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message":      "User added to group successfully",
		"participants": conversation.Participants,
	})
}

// Group member leaves group
func LeaveGroup(c *fiber.Ctx) error {
	type ReqBody struct {
		UserID         uint `json:"user_id"`
		ConversationID uint `json:"conversation_id"`
	}

	var reqBody ReqBody
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Delete the user from the conversation
	if err := initializer.DB.Table("conversation_users").Where("conversation_id = ? AND user_id = ?", reqBody.ConversationID, reqBody.UserID).Delete(nil).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to leave conversation"})
	}

	//create conversation object
	var conversation models.Conversation

	// Retrieve the conversation from the database
	if err := initializer.DB.First(&conversation, reqBody.ConversationID).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get participant from Database",
		})
	}

	// Preload participants
	if err := initializer.DB.Preload("Participants").First(&conversation, reqBody.ConversationID).Error; err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to preload conversation participants",
		})
	}

	// return conversation.participants
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message":      "User left conversation successfully",
		"Conversation": conversation.Participants,
	})
}

func GetAllGroups(context *fiber.Ctx) error {
	// Get user ID from token
	user, ok := context.Locals("user").(models.User)
	if !ok {
		return context.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve user from context"})
	}
	userID := user.ID

	// Fetch all groups associated with the user ID
	var groups []models.Group
	if err := initializer.DB.
		Preload("GroupConversation").
		Preload("GroupConversation.Participants").
		Preload("GroupConversation.Messages").
		Preload("GroupConversation.Messages.Sender").
		Where("admin_id = ? OR conversation_id IN (SELECT conversation_id FROM conversation_users WHERE user_id = ?)", userID, userID).
		Find(&groups).Error; err != nil {
		return context.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Send all the groups associated with the user
	return context.Status(http.StatusOK).JSON(
		fiber.Map{"groups": groups})
}

// Group Admin can remove a user from the group
func RemoveGroupMember(c *fiber.Ctx) error {
	//create req body
	type ReqBody struct {
		ConversationID uint `json:"conversation_id"`
		GrpAdminId     uint `json:"grp_admin_id"`
		RemoveUserId   uint `json:"delete_user_id"`
	}

	//parse request body
	var reqBody ReqBody
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// check admin id is available in the group table for respective conversation id
	var group models.Group
	if err := initializer.DB.Where("conversation_id = ?", reqBody.ConversationID).Find(&group).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to find the Group Object",
		})
	}

	//if admin id does not exist in the group table then throw an error
	if group.AdminId != reqBody.GrpAdminId {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "You are not Group Admin!!!You are not authorized to delete group users",
		})
	}

	// if admin id exists then delete the user_id from conversation_user table for respective conversation id
	if err := initializer.DB.Table("conversation_users").Where("conversation_id =? AND user_id =?", reqBody.ConversationID, reqBody.RemoveUserId).Delete(nil).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove user from the group",
		})
	}

	//create conversation object
	var conversation models.Conversation

	// Retrieve the conversation from the database using conversation id
	if err := initializer.DB.First(&conversation, reqBody.ConversationID).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error ": "Failed to load conversation from database",
		})
	}

	// Preload participants
	if err := initializer.DB.Preload("Participants").First(&conversation, reqBody.ConversationID).Error; err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to preload conversation participants",
		})
	}

	// return conversation.participants
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message":                   "User left conversation successfully",
		"conversation_participants": conversation.Participants,
	})

}

// Group admin can delete a Group
func DeleteGroup(c *fiber.Ctx) error {

	//get conversation id and userId from request
	type ReqBody struct {
		ConversationID uint `json:"conversation_id"`
		GrpAdminId     uint `json:"grp_admin_id"`
	}

	var reqBody ReqBody
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	// check admin id is available in the group table for respective conversation id
	var group models.Group
	if err := initializer.DB.Where("conversation_id = ?", reqBody.ConversationID).Find(&group).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to find the Group Object",
		})
	}
	//check if the user is admin or not
	if group.AdminId != reqBody.GrpAdminId {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "You are not Group Admin!!!You are not authorized to delete the Group",
		})
	}

	//delete the group and its related participants from groupConversation.participants[] from the database
	if err := initializer.DB.Table("conversation_users").Where("conversation_id =?", reqBody.ConversationID).Delete(nil).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete group",
		})
	}

	if err := initializer.DB.Delete(&group).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete the group",
		})
	}

	//create the new group list and send it from response
	var groups []models.Group
	if err := initializer.DB.Preload("GroupConversation").Preload("GroupConversation.Participants").Where("admin_id =? OR conversation_id IN (SELECT conversation_id FROM conversation_users WHERE user_id =?)", reqBody.GrpAdminId, reqBody.GrpAdminId).Find(&groups).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(groups)

}

func IsGroupAndConversation(c *fiber.Ctx) error {
	type Conversation struct {
		ConversationID uint `json:"conversation_id"`
	}

	var reqBody Conversation
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Check if ConversationID exists in the group table
	var isGroup bool
	var group models.Group
	if err := initializer.DB.Where("conversation_id = ?", reqBody.ConversationID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			isGroup = false
		} else {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to query group table"})
		}
	} else {
		isGroup = true
	}

	// Retrieve the conversation from the database using conversation id
	var conversation models.Conversation
	if err := initializer.DB.Preload("Participants").First(&conversation, reqBody.ConversationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Conversation not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load conversation from database"})
	}

	response := fiber.Map{
		"isGroup":      isGroup,
		"Conversation": conversation,
	}

	// If it's a group conversation, include the group object in the response
	if isGroup {
		response["group"] = group
	}

	return c.Status(http.StatusOK).JSON(response)
}
