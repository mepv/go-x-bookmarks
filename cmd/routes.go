package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mepv/go-x-bookmarks/internal/config"
	"github.com/mepv/go-x-bookmarks/internal/handlers"
	"net/http"
)

func routes(app *config.AppConfig) http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)
	mux.Use(NoSurf)
	mux.Use(SessionLoad)

	mux.Get("/api/v1/bookmarks/oauth/authorize", handlers.BuildAuthorizationUrl)
	mux.Get("/api/v1/bookmarks/oauth/callback", handlers.ExchangeCodeForToken)
	mux.Get("/api/v1/bookmarks/users/{username}", handlers.UserHandler)

	return mux
}
