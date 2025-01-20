package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/mepv/go-x-bookmarks/internal/config"
	"github.com/mepv/go-x-bookmarks/internal/helpers"
	"github.com/mepv/go-x-bookmarks/internal/models"
	"github.com/mepv/go-x-bookmarks/internal/render"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var app *config.AppConfig

func NewHandlers(a *config.AppConfig) {
	app = a
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	err := render.Template(w, r, "home.page.gohtml", &models.TemplateData{})
	if err != nil {
		log.Printf("Error rendering template: %v", err)
		helpers.ServerError(w, err)
		return
	}
}

func BookmarkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Form parse error: %v", err)
		helpers.ClientError(w, http.StatusBadRequest)
		return
	}

	accessToken, ok := app.Session.Get(r.Context(), "access_token").(string)
	if !ok {
		app.Session.Put(r.Context(), "error", accessTokenNotFound)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		log.Printf("access token not found")
		return
	}

	username := r.Form.Get("username")
	userResp, err := fetchUserInformation(accessToken, username)
	if err != nil {
		log.Printf("Error fetching user information: %v", err)
		helpers.ServerError(w, err)
		return
	}

	bookmarks, err := fetchBookmarks(accessToken, userResp.Data)
	if err != nil {
		log.Printf("Error fetching bookmarks: %v", err)
		helpers.ServerError(w, err)
		return
	}

	data := make(map[string]interface{})
	data["bookmarks"] = bookmarks
	data["username"] = userResp.Data.Username
	err = render.Template(w, r, "bookmarks.page.gohtml", &models.TemplateData{
		Data: data,
	})
	if err != nil {
		log.Printf("Error rendering template: %v", err)
		helpers.ServerError(w, err)
		return
	}
}

func fetchUserInformation(accessToken string, username string) (*models.UserResponse, error) {
	userInfoUri := os.Getenv("USER_INFORMATION_URI")
	apiURL := fmt.Sprintf(userInfoUri, username)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create API request to fetch user information: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send API request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			//
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	var userResp models.UserResponse
	err = json.Unmarshal(body, &userResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API response JSON: %w", err)
	}
	return &userResp, nil
}

func fetchBookmarks(accessToken string, user models.User) ([]models.Bookmark, error) {
	bookmarksUri := os.Getenv("BOOKMARKS_URI")
	apiURL := fmt.Sprintf(bookmarksUri, user.ID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create API request to fetch user bookmarks: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send bookmark request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("failed to close body read: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	var bookmarkResp models.BookmarksResponse
	err = json.Unmarshal(body, &bookmarkResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API response JSON: %w", err)
	}
	return bookmarkResp.Data, nil
}
