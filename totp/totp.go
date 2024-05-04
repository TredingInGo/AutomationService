package totp

import (
	"log"
	"time"

	"github.com/pquerna/otp/totp"
)

func GetTOPT(clientCode string) string {
	secret := "PV5J747OQSTMGEN2NVYSH7IRYM"
	if clientCode == "A55697181" {
		secret = "NZ2VJ25KDF5LFQCSXWNC6EE5K4"
	}

	// Generate a TOTP token using the current time
	token, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		log.Println("Error generating TOTP:", err)
		return ""
	}

	log.Printf("Current TOTP for %v = : %s\n", secret, token)

	return token
}
