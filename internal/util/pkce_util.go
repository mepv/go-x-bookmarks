package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log"
)

func GenerateCodeVerifier() (string, error) {
	length := 64
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

	verifier := make([]byte, length)

	_, err := rand.Read(verifier)
	if err != nil {
		log.Fatalf("Failed to generate verifier: %v", err)
		return "", err
	}

	for i := 0; i < length; i++ {
		verifier[i] = charset[int(verifier[i])%len(charset)]
	}

	return string(verifier), nil
}

func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
