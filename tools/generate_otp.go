package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pquerna/otp/totp"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <base32-secret>", os.Args[0])
	}
	secret := os.Args[1]
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(code)
}
