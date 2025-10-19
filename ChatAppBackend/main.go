package main

import (
	"ChatApp/initializer"
	"ChatApp/routes"
	"ChatApp/wshandler"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

func (r *Repository) SetupRoutes(app *fiber.App) {

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOrigins:     "http://localhost:4200",
		AllowMethods:     "GET,POST,PUT,DELETE",
		AllowHeaders:     "Content-Type,Authorization", // Allow credentials (cookies, authorization headers, etc.)
	}))
	routes.AdminRoutes(app)
	routes.UserRoutes(app)

}

func init() {

	fmt.Println("start init")
	initializer.LoadEnvVar()
	initializer.ConnectToDB()
	initializer.SyncDatabase()
	initializer.ImageKitObject()
	fmt.Println("end init")
}

func main() {

	fmt.Println("start main")

	r := Repository{
		DB: initializer.DB,
	}
	app := fiber.New()
	app.Get("/ws/:conversationId", websocket.New(wshandler.WsConversationHandler))

	r.SetupRoutes(app)
	if err := app.Listen(":8080"); err != nil {
		log.Fatal(err)
	}
}
