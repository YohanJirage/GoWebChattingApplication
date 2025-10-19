package controller

import (
	"ChatApp/initializer"
	"ChatApp/models"
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/imagekit-developer/imagekit-go/api/uploader"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ctx = context.Background()

func Validate(context *fiber.Ctx) error {

	user := context.Locals("user")

	context.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login Successful",
		"user":    user,
	})
	return nil
}

func generateOTP(length int) string {
	charset := "0123456789"
	otp := make([]byte, length)
	_, err := rand.Read(otp)
	if err != nil {
		panic(err)
	}
	for i := range otp {
		otp[i] = charset[int(otp[i])%len(charset)]
	}
	return string(otp)
}

func sendEmail(from, password, to, subject, body string) error {
	auth := smtp.PlainAuth("", from, password, "smtp.gmail.com")

	msg := []byte("From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		body)

	err := smtp.SendMail("smtp.gmail.com:587", auth, from, []string{to}, msg)
	if err != nil {
		return err
	}
	return nil
}

func OTPSendToEmail(c *fiber.Ctx) error {
	var bodyStruct struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&bodyStruct); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to read body"})
	}
	otp := generateOTP(6)

	// Sender email credentials
	from := "yjirage@gmail.com"
	password := "xhfvxrbonhyxuezm"

	// Recipient email
	to := bodyStruct.Email

	// Email content
	subject := "One-Time Password (OTP)"
	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Email Verification</title>
	</head>
	<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
		<div style="max-width: 600px; margin: 0 auto; background-color: #fff; padding: 20px; border-radius: 10px; box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);">
			<h2 style="color: #007bff; text-align: center;">BaatCheet Email Verification</h2>
			<p style="text-align: center;">Please use the following OTP to verify your email address:</p>
			<div style="background-color: #007bff; color: #fff; text-align: center; font-size: 24px; padding: 10px; border-radius: 5px; margin: 20px auto;">
				%s
			</div>
			<p style="text-align: center;">If you didn't request this verification, please ignore this email.</p>
		</div>
	</body>
	</html>
	`, otp)

	// Send email
	err := sendEmail(from, password, to, subject, body)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to Send Email"})
	}
	fmt.Println(otp)
	var emailOtp models.EmailOTP

	// Genrate Otp and save temprory in database
	emailOtp.Email = bodyStruct.Email
	emailOtp.OTP = otp

	result := initializer.DB.Create(&emailOtp)
	if result.Error != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to create user", "detail": result.Error.Error()})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"Message": "OTP send Successfullt",
		"email": bodyStruct.Email,
	})
}

func OTPVerify(c *fiber.Ctx) error {
	var receivedBody models.EmailOTP
	if err := c.BodyParser(&receivedBody); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to read body"})
	}

	var emailOtp models.EmailOTP
	if err := initializer.DB.Where("email = ?", receivedBody.Email).First(&emailOtp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid email or OTP"})
		}
	}

	// OTP verification succeeded
	if emailOtp.Email == receivedBody.Email && emailOtp.OTP == receivedBody.OTP {
		// Delete the OTP record
		if err := initializer.DB.Delete(&emailOtp).Error; err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete OTP record"})
		}
		return c.Status(http.StatusOK).JSON(fiber.Map{"message": "OTP Verified", "email": emailOtp.Email, "isVerified": true})
	}

	// OTP verification failed delete the emailOtp entry from the database
	if err := initializer.DB.Delete(&emailOtp).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete OTP record"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "OTP Mismatch", "email": emailOtp.Email, "isVerified": false})
}

// SignUp registers a new user
func SignUp(c *fiber.Ctx) error {

	// Parse the request body
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	username := form.Value["username"][0]
	email := form.Value["email"][0]
	phone := form.Value["phone"][0]
	about := form.Value["about"][0]
	password := form.Value["password"][0]

	// Check if the email is already registered
	var existingUser models.User
	if err := initializer.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		// Email already exists, return an error response
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "User Already Registered'"})
	}

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	// Create the user model
	var user models.User
	user.Username = username
	user.Email = email
	user.Phone = phone
	user.About = about
	user.Password = string(hash)

	// Upload the profile picture
	file, err := form.File["profilePicture"][0].Open()
	if err != nil {
		return err
	}
	defer file.Close()

	extension := filepath.Ext(form.File["profilePicture"][0].Filename)

	newFilename := fmt.Sprintf("%s%s", email, extension)

	uploadResp, err := initializer.Ik.Uploader.Upload(ctx, file, uploader.UploadParam{
		FileName: newFilename,
	})
	if err != nil {
		fmt.Println("Error uploading image:", err)

	}

	user.ProfilePicture = uploadResp.Data.Url

	// Save the user to the database
	result := initializer.DB.Create(&user)
	if result.Error != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to create user", "detail": result.Error.Error()})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Account Created Successfully", "user": user})
}

func Login(c *fiber.Ctx) error {
	fmt.Println("In Login")
	// Get email/pass
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to read body"})
	}

	// Look up requested user
	var user models.User
	initializer.DB.First(&user, "email = ?", body.Email)

	if user.ID == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "User not found. Please sign up to login"})
	}

	// Compare sent in password with saved user password hash
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid password"})
	}

	// Generate a JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":     user.ID,
		"email":   user.Email,
		"isAdmin": user.IsAdmin,
		"exp":     time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to create token"})
	}

	// Send it back
	cookie := fiber.Cookie{
		Name:     "Authorization",
		Value:    tokenString,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: false,
	}

	c.Cookie(&cookie)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Logged in successfully",
		"user":    user,
	})
}

func ChangePassword(c *fiber.Ctx) error {

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to read body"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	newPasshash := string(hash)
	updateErr := initializer.DB.Model(&models.User{}).Where("email =?", body.Email).Update("password", newPasshash).Error

	if updateErr != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Failed to Update Password", "error": updateErr.Error()})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Successfully Update Password"})
}

// update user information
func UpdateUser(c *fiber.Ctx) error {

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	// user_id := form.Value["userId"][0]
	email := form.Value["email"][0]
	username := form.Value["username"][0]
	phone := form.Value["phone"][0]
	about := form.Value["about"][0]

	var user models.User
	user.Username = username
	user.Phone = phone
	user.About = about
	user.Email = email

	user.ID = c.Locals("user").(models.User).ID

	// Handle profile picture
	if len(form.File["profilePicture"]) > 0 {
		file, err := form.File["profilePicture"][0].Open()

		// Open the uploaded profile picture file
		if err != nil {
			return err
		}
		defer file.Close()

		extension := filepath.Ext(form.File["profilePicture"][0].Filename)

		newFilename := fmt.Sprintf("%s%s", email, extension)

		uploadResp, err := initializer.Ik.Uploader.Upload(ctx, file, uploader.UploadParam{
			FileName: newFilename,
		})
		if err != nil {
			fmt.Println("Error uploading image:", err)

		}
		user.ProfilePicture = uploadResp.Data.Url
	}

	// Update user information in the database
	result := initializer.DB.Model(&user).Select("username", "phone", "about", "profile_picture").Updates(&user)
	if result.Error != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to update user", "detail": result.Error.Error()})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Profile updated successfully", "user": user})
}
