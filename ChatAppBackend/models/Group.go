package models

import "gorm.io/gorm"

type Group struct {
	gorm.Model
	GroupName         string       `json:"groupname"`
	ConversationID    uint         `json:"conversation_id"`
	GroupConversation Conversation `json:"group_conversation,omitempty" gorm:"foreignKey:ConversationID"`
	AdminId           uint         `json:"admin_id"`
	Admin             User         `json:"admin,omitempty" gorm:"foreignKey:AdminId"`
}
