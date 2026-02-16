package config

import (
	"database/sql"
	"html/template"
	"log"
	"os"

	"github.com/alexedwards/scs/v2"
)

type AppConfig struct {
	UseCache      bool
	TemplateCache map[string]*template.Template
	InfoLog       *log.Logger
	ErrorLog      *log.Logger
	InProduction  bool
	Session       *scs.SessionManager
	DB            *sql.DB
}

type EnvConfig struct {
	ClientId           string
	ClientSecret       string
	RedirectUri        string
	AuthorizationUri   string
	TokenUri           string
	UserInformationUri string
	BookmarksUri       string
	Scope              string
	MaxResults         string
	DatabaseURL        string
	DatabaseDriver     string
}

func NewEnvConfig() *EnvConfig {
	return &EnvConfig{
		ClientId:           getEnv("CLIENT_ID", ""),
		ClientSecret:       getEnv("CLIENT_SECRET", ""),
		RedirectUri:        getEnv("REDIRECT_URI", ""),
		AuthorizationUri:   getEnv("AUTHORIZATION_URI", ""),
		TokenUri:           getEnv("TOKEN_URI", ""),
		UserInformationUri: getEnv("USER_INFORMATION_URI", ""),
		BookmarksUri:       getEnv("BOOKMARKS_URI", ""),
		Scope:              getEnv("SCOPE", ""),
		MaxResults:         getEnv("MAX_RESULTS", "10"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		DatabaseDriver:     getEnv("DATABASE_DRIVER", "postgres"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
