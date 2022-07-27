package util

import (
	"github.com/ethereum/go-ethereum/console/prompt"
	"golang.org/x/xerrors"
)

func GetPassword(confirmation bool) (string, error) {
	password, err := prompt.Stdin.PromptPassword("Password: ")
	if err != nil {
		return "", err
	}

	if confirmation {
		confirm, err := prompt.Stdin.PromptPassword("Repeat password: ")
		if err != nil {
			return "", xerrors.Errorf("Failed to read password confirmation: %v", err)
		}
		if password != confirm {
			return "", xerrors.New("Passwords do not match")
		}
	}

	return password, nil
}
