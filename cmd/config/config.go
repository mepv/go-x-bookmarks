package config

import "os"

type Config struct {
	ClientId         string
	ClientSecret     string
	RedirectUri      string
	AuthorizationUri string
	TokenUri         string
	Scope            string
}

func NewConfig() *Config {
	return &Config{
		ClientId:         getEnv("CLIENT_ID", ""),
		ClientSecret:     getEnv("CLIENT_SECRET", ""),
		RedirectUri:      getEnv("REDIRECT_URI", ""),
		AuthorizationUri: getEnv("AUTHORIZATION_URI", ""),
		TokenUri:         getEnv("TOKEN_URI", ""),
		Scope:            getEnv("SCOPE", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
