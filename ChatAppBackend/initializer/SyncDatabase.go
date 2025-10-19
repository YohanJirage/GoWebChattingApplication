package initializer

import (
	"ChatApp/models"
	"fmt"
	"log"
)

func SyncDatabase() {

	fmt.Println("In Migration")
	if DB == nil {
		log.Fatal("DB instance is nil")
	}

	fmt.Println(DB)
	DB.AutoMigrate(&models.User{})
	DB.AutoMigrate(&models.UserStatus{})
	DB.AutoMigrate(&models.Message{})
	DB.AutoMigrate(&models.Group{})
	DB.AutoMigrate(&models.Conversation{})
	DB.AutoMigrate(&models.EmailOTP{})
	fmt.Print("end Migration")
}
