package models

import (
	"gorm.io/gorm"
)

type EmailOTP struct {
	gorm.Model
	Email string `json:"email"`
	OTP   string `json:"otp"`
}
