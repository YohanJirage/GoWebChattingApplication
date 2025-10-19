package models

import (
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	SenderID int    `json:"sender_id"`
	Sender   User   `json:"sender,omitempty" gorm:"foreignKey:SenderID"`
	Image    string `json:"image"`
	Video    string `json:"video"`
	Audio    string `json:"audio"`
	PDF      string `json:"pdf"`
	Content  string `json:"message"`
}

type ReceivedMessage struct {
	ConversationID int         `json:"conversation_id"`
	SenderID       int         `json:"sender_id"`
	ContentType    string      `json:"content_type"`
	Content        interface{} `json:"content"`
}

func MigrateMessage(db *gorm.DB) error {
	err := db.AutoMigrate(&Message{})
	return err
}
