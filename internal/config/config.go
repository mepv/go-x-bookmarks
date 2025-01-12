package config

import (
	"github.com/alexedwards/scs/v2"
	"log"
	"os"
)

type AppConfig struct {
	UseCache bool
	//TemplateCache map[string]*template.Template
	InfoLog      *log.Logger
	ErrorLog     *log.Logger
	InProduction bool
	Session      *scs.SessionManager
}

type EnvConfig struct {
	ClientId         string
	ClientSecret     string
	RedirectUri      string
	AuthorizationUri string
	TokenUri         string
	Scope            string
}

func NewEnvConfig() *EnvConfig {
	return &EnvConfig{
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
