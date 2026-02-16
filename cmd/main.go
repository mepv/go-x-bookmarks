package main

import (
	"context"
	"database/sql"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
	"github.com/mepv/go-x-bookmarks/internal/config"
	"github.com/mepv/go-x-bookmarks/internal/database"
	"github.com/mepv/go-x-bookmarks/internal/handlers"
	"github.com/mepv/go-x-bookmarks/internal/helpers"
	"github.com/mepv/go-x-bookmarks/internal/models"
	"github.com/mepv/go-x-bookmarks/internal/render"
)

const portNumber = ":8080"

var app config.AppConfig
var session *scs.SessionManager
var infoLog *log.Logger
var errorLog *log.Logger

func main() {
	// Load environment variables from .env file before startup.
	err := godotenv.Load("../env-var/.env")
	if err != nil {
		log.Printf("Warning: unable to load .env file: %v", err)
	}

	// Setup
	err = run()
	if err != nil {
		log.Fatal(err)
	}

	if app.DB != nil {
		defer func(DB *sql.DB) {
			err := DB.Close()
			if err != nil {
				log.Printf("Error closing database connection: %v", err)
			}
		}(app.DB)
	}

	log.Printf("Server starting on port %s...", portNumber)
	srv := &http.Server{
		Addr:    portNumber,
		Handler: routes(&app),
	}
	err = srv.ListenAndServe()
	log.Fatal(err)
}

func run() error {
	// What is going to put in the session
	gob.Register(models.TokenResponse{})
	gob.Register(models.User{})
	gob.Register(models.UserResponse{})
	gob.Register(models.Bookmark{})

	// Read flags
	inProductionMode := flag.Bool("production", false, "Application is running in production mode")
	useCache := flag.Bool("cache", true, "Use template cache")
	flag.Parse()

	// change this to true when in production
	app.InProduction = *inProductionMode
	app.UseCache = *useCache

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	app.InfoLog = infoLog

	errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	app.ErrorLog = errorLog

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction

	app.Session = session

	// Initialize env configuration
	envCfg := config.NewEnvConfig()

	if envCfg.DatabaseURL != "" {
		if !sqlDriverAvailable(envCfg.DatabaseDriver) {
			log.Printf("DATABASE_DRIVER=%s is not registered. Add a Postgres SQL driver import to enable DB connectivity.", envCfg.DatabaseDriver)
		} else {
			dbPool, err := database.ConnectPostgres(context.Background(), envCfg.DatabaseDriver, envCfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("failed to initialize postgres: %w", err)
			}
			app.DB = dbPool
			log.Printf("Connected to Postgres successfully")
		}
	} else {
		log.Printf("DATABASE_URL not configured: Postgres repositories are disabled")
	}

	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("cannot create template cache")
		return err
	}
	app.TemplateCache = tc
	handlers.NewHandlers(&app)
	handlers.NewAuthorizationHandlers(&app)
	render.NewRenderer(&app)
	helpers.NewHelpers(&app)

	return nil
}

func sqlDriverAvailable(name string) bool {
	for _, driver := range sql.Drivers() {
		if driver == name {
			return true
		}
	}
	return false
}
