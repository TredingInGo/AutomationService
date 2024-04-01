package totp

import (
	"fmt"
	"time"

	"github.com/pquerna/otp/totp"
)

func GetTOPT() string {
	secret := "PV5J747OQSTMGEN2NVYSH7IRYM"

	// Generate a TOTP token using the current time
	token, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		fmt.Println("Error generating TOTP:", err)
		return ""
	}

	fmt.Printf("Current TOTP: %s\n", token)

	return token
}
