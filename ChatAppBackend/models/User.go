package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username       string `json:"username"`
	Password       string `json:"password"`
	Email          string `gorm:"unique" json:"email"`
	ProfilePicture string `json:"profile_picture" `
	Phone          string `json:"phone"`
	About          string `json:"about"`
	IsAdmin        bool   `json:"is_admin"`
}

func MigrateUser(db *gorm.DB) error {
	err := db.AutoMigrate(&User{})
	return err
}
