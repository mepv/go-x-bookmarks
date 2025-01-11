package routes

import (
	"github.com/mepv/go-x-bookmarks/internal/handlers"
	"net/http"
)

func SetupRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/bookmarks/oauth/", handlers.AuthorizeHandler)

	return mux
}
