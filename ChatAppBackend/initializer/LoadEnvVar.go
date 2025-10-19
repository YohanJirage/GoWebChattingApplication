package initializer

import (
	"log"
	"os"

	"github.com/imagekit-developer/imagekit-go"
	"github.com/joho/godotenv"
)

func LoadEnvVar() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

var Ik *imagekit.ImageKit

//ImageKitObject initializes the ImageKit client with credentials and endpoint URL.
func ImageKitObject() {

	privateKey := os.Getenv("IMAGEKIT_PRIVATE_KEY")
	publicKey := os.Getenv("IMAGEKIT_PUBLIC_KEY")
	endpointURL := os.Getenv("IMAGEKIT_ENDPOINT_URL")

	// Create a new ImageKit client instance with the retrieved credentials
	ik := imagekit.NewFromParams(imagekit.NewParams{
		PrivateKey:  privateKey,
		PublicKey:   publicKey,
		UrlEndpoint: endpointURL,
	})

	Ik = ik

}
