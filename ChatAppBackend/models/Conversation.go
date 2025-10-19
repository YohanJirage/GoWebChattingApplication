package models

import "gorm.io/gorm"

// GORM MODEL TYPE
type Conversation struct {
	gorm.Model
	Participants []User    `gorm:"many2many:conversation_users;" json:"participants"`
	Messages     []Message `gorm:"many2many:conversation_messages;" json:"messages"`
}

// CUSTOME MODEL
type CreateConversation struct {
	CreaterId    uint  `json:"UserId"`
	Participants []int `json:"Participants"`
}

type SendConversation struct {
	Participants []User    `json:"participants"`
	Message      []Message `json:"messages"`
	IsGroup      bool      `json:"isGroup"`
	Group        Group     `json:"group"`
}
