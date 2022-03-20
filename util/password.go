package util

import (
	"fmt"
	"github.com/ethereum/go-ethereum/console/prompt"
)

func GetPassword() string {
	password, err := prompt.Stdin.PromptPassword("Password: ")
	if err != nil {
		fmt.Printf("Failed to read password: %v", err)
		return ""
	}

	return password
}
