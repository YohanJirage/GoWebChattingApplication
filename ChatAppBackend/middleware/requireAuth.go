package middleware

import (
	"ChatApp/initializer"
	"ChatApp/models"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

func RequireAuth(c *fiber.Ctx) error {
	fmt.Println("In Auth")

	// Get the cookie value from the request header
	tokenString := c.Get("Authorization")

	if tokenString == "" {
		return c.Status(http.StatusUnauthorized).SendString("Unauthorized1")
	}

	// Parse and validate the JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"]) // HS256 algorithm
		}

		// Return the secret key to validate the token
		return []byte(os.Getenv("SECRET")), nil
	})

	// Check for errors during token parsing and validation
	if err != nil || token == nil || !token.Valid {
		return c.Status(http.StatusUnauthorized).SendString("Unauthorized2")
	}

	// Extract claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(http.StatusUnauthorized).SendString("Unauthorized3")
	}

	// Check the expiration time of the token
	exp, ok := claims["exp"].(float64)
	if !ok || float64(time.Now().Unix()) > exp {
		return c.Status(http.StatusUnauthorized).SendString("Unauthorized4")
	}

	// Find the user with the token subject (sub)
	var user models.User
	initializer.DB.First(&user, claims["sub"])
	if user.ID == 0 {
		return c.Status(http.StatusUnauthorized).SendString("Unauthorized5")
	}

	// Attach the user object to the request context
	c.Locals("user", user)

	// Continue processing the request
	return c.Next()
}
