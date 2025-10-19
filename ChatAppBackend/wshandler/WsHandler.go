package wshandler

import (
	"ChatApp/initializer"
	"ChatApp/models"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/imagekit-developer/imagekit-go/api/uploader"
)

// Declare a background context
var ctx = context.Background()

// Define a map to store connected clients per room/conversation
var clients = make(map[string]map[*websocket.Conn]bool)

// Mutex to synchronize access to the clients map
var clientsMutex sync.Mutex

// Struct to represent the metadata of the message being sent
type SendMessageMetadata struct {
	ConversationID int    `json:"conversation_id"`
	SenderID       int    `json:"sender_id"`
	Message        string `json:"message"`
	ContentType    string `json:"content_type"`
	File_extension string `json:"file_extension"`
}

// Struct to represent the data being sent over the WebSocket connection
type Data struct {
	Metadat  SendMessageMetadata `json:"metadata"`
	FileData []byte              `json:"fileData"`
}

// Handler function for WebSocket connections
func WsConversationHandler(c *websocket.Conn) {

	conversationId := c.Params("conversationId")
	clientsMutex.Lock() // Lock the clients mutex
	// Create a new entry in the clients map if it doesn't exist
	if _, ok := clients[conversationId]; !ok {
		clients[conversationId] = make(map[*websocket.Conn]bool)
	}
	// Add the current client connection to the clients map for the specified conversation
	clients[conversationId][c] = true
	clientsMutex.Unlock()

	defer func() {
		// Remove the current client connection from the clients map
		delete(clients[conversationId], c)
		// If there are no more clients in the conversation, delete the conversation entry from the clients map
		if len(clients[conversationId]) == 0 {
			delete(clients, conversationId)
		}
		// Close the WebSocket connection
		c.Close()
	}()

	existingMessages := fetchExistingMessages(conversationId)

	// Send existing messages to the client
	clientsInRoom, ok := clients[conversationId]
	if !ok {
		log.Printf("No clients in room %s\n", conversationId)
		return
	}

	for client := range clientsInRoom {
		if err := client.WriteJSON(existingMessages); err != nil {
			log.Println("Error broadcasting message to client:", err)
			log.Printf("Failed to write message to client %v\n", client.RemoteAddr())
		}
	}

	// Continuously read messages from the client
	for {

		var data Data
		err := c.ReadJSON(&data)
		if err != nil {
			fmt.Println("Error reading data:", err)
			break
		}

		fmt.Println("in data")

		// Process the received data
		metadata := data.Metadat
		fmt.Println("meta:", metadata)
		fileData := data.FileData
		var bytes []byte = fileData
		allMessages := saveMessageMedia(metadata, bytes)

		// Broadcast message to all clients in the same room
		broadcastMessage(allMessages, conversationId)
	}
}

// Function to save message media to storage and database
func saveMessageMedia(metaData SendMessageMetadata, fileData []byte) []models.Message {

	var message models.Message
	message.SenderID = metaData.SenderID
	message.Content = metaData.Message

	switch metaData.ContentType {
	case "image":

		imagePath, erro := saveFile(fileData, "image", metaData.File_extension)
		if erro != nil {
			fmt.Println("Error saving image:", erro)
			break
		}
		message.Image = imagePath

	case "video":
		videoPath, erro := saveFile(fileData, "video", metaData.File_extension)
		if erro != nil {
			fmt.Println("Error saving video:", erro)
			break
		}
		message.Video = videoPath

	case "audio":
		audioPath, erro := saveFile(fileData, "audio", metaData.File_extension)
		if erro != nil {
			fmt.Println("Error saving audio:", erro)
		}
		message.Audio = audioPath

	case "pdf":
		pdfPath, erro := saveFile(fileData, "pdf", metaData.File_extension)
		if erro != nil {
			fmt.Println("Error saving pdf:", erro)
		}
		message.PDF = pdfPath

	}

	// Save the message to the database
	if err := initializer.DB.Create(&message).Error; err != nil {
		log.Println("Error saving message:", err)
		return nil
	}

	// Fetch the conversation associated with the message
	var conversation models.Conversation
	if err := initializer.DB.Preload("Messages").First(&conversation, metaData.ConversationID).Error; err != nil {
		log.Println("Error fetching conversation:", err)
	}

	// Associate the message with the conversation
	conversation.Messages = append(conversation.Messages, message)

	// Save the updated conversation
	if err := initializer.DB.Save(&conversation).Error; err != nil {
		fmt.Println("Error to append message", err)
	}

	if err := initializer.DB.Preload("Messages.Sender").First(&conversation, metaData.ConversationID).Error; err != nil {
		fmt.Println("Error to poppulate Messages", err)
	}
	return conversation.Messages
}

// Function to save file to storage and return its URL
func saveFile(fileData []byte, contentType string, file_extension string) (string, error) {
	// Construct the new filename using a unique identifier and the content type extension
	newFilename := fmt.Sprintf("%s%s.%s", contentType, uuid.New().String(), file_extension)

	tempFile, err := os.Create(newFilename)

	if err != nil {
		return "", fmt.Errorf("error creating temporary file: %w", err)
	}
	defer os.Remove(newFilename) // Clean up the temporary file after it's used

	if _, err := tempFile.Write(fileData); err != nil {
		return "", fmt.Errorf("error writing to temporary file: %w", err)
	}
	tempFile.Close()
	file, err := os.Open(newFilename)
	if err != nil {
		return "", fmt.Errorf("error writing to temporary file: %w", err)
	}
	defer file.Chdir()

	// Upload the file using your uploader
	uploadResp, err := initializer.Ik.Uploader.Upload(ctx, file, uploader.UploadParam{
		FileName: newFilename, // Assuming FileName field requires the path of the file
	})
	if err != nil {
		return "", err
	}

	return uploadResp.Data.Url, nil

}

// Function to broadcast message to all clients in a conversation
func broadcastMessage(msg []models.Message, conversationId string) {

	clientsMutex.Lock()
	defer clientsMutex.Unlock() // Unlock the mutex when done with the critical section

	clientsInRoom, ok := clients[conversationId]
	if !ok {
		log.Printf("No clients in room %s\n", conversationId)
		return
	}

	// Iterate over each client in the conversation and send the message
	for client := range clientsInRoom {
		if err := client.WriteJSON(msg); err != nil {
			log.Println("Error broadcasting message to client:", err)
			log.Printf("Failed to write message to client %v\n", client.RemoteAddr())
		}
	}
}

// Function to fetch existing messages for a conversation
func fetchExistingMessages(conversationId string) []models.Message {

	convId, _ := strconv.Atoi(conversationId)

	var conversation models.Conversation
	if err := initializer.DB.Preload("Messages.Sender").First(&conversation, convId).Error; err != nil {
		fmt.Println("Error to poppulate Messages", err)
	}
	return conversation.Messages

}
