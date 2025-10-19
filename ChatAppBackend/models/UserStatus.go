package models

import "gorm.io/gorm"

type UserStatus struct {
	gorm.Model
	UserID     int    `json:"uid"`
	User       User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	IsOnline   bool   `json:"is_online"`
	LastOnline string `json:"last_online,omitempty"`
}
