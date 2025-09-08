package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mepv/go-x-bookmarks/internal/config"
	"github.com/mepv/go-x-bookmarks/internal/helpers"
	"github.com/mepv/go-x-bookmarks/internal/models"
	"github.com/mepv/go-x-bookmarks/internal/render"
	"github.com/mepv/go-x-bookmarks/internal/storage"
)

var app *config.AppConfig
var bookmarkStorage *storage.BookmarkStorage

func NewHandlers(a *config.AppConfig) {
	app = a
	// Initialize bookmark storage to save in data/bookmarks.txt
	bookmarkStorage = storage.NewBookmarkStorage("data/bookmarks.txt")
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

	username := r.Form.Get("username")

	// Always fetch from API and merge with stored data
	accessToken, ok := app.Session.Get(r.Context(), "access_token").(string)
	if !ok {
		app.Session.Put(r.Context(), "error", accessTokenNotFound)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		log.Printf("access token not found")
		return
	}

	userResp, err := fetchUserInformation(accessToken, username)
	if err != nil {
		log.Printf("Error fetching user information: %v", err)
		helpers.ServerError(w, err)
		return
	}

	// Fetch fresh bookmarks from API
	apiBookmarks, err := fetchBookmarks(accessToken, userResp.Data)
	if err != nil {
		log.Printf("Error fetching bookmarks from API: %v", err)
		helpers.ServerError(w, err)
		return
	}

	// Load existing bookmarks from storage
	storedBookmarks, err := bookmarkStorage.LoadBookmarks()
	if err != nil {
		log.Printf("Error loading stored bookmarks: %v", err)
		storedBookmarks = nil // Continue with empty stored bookmarks
	}

	// Merge API bookmarks with stored bookmarks (deduplicate)
	bookmarks := mergeBookmarks(storedBookmarks, apiBookmarks, username)

	// Save merged bookmarks to storage
	err = bookmarkStorage.SaveBookmarks(username, bookmarks)
	if err != nil {
		log.Printf("Error saving merged bookmarks to storage: %v", err)
	} else {
		log.Printf("Saved %d merged bookmarks to storage for user %s", len(bookmarks), username)
	}

	data := make(map[string]interface{})
	data["bookmarks"] = bookmarks
	data["username"] = username
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
	maxResults := os.Getenv("MAX_RESULTS")
	// todo: fixed retrieving setup, it shouldn't set the max again, using the os.Getenv instead the config struct
	if maxResults == "" {
		maxResults = "10"
	}

	// Add max_results query parameter
	apiURL := fmt.Sprintf(bookmarksUri, user.ID)
	if maxResults != "" {
		if strings.Contains(apiURL, "?") {
			apiURL += "&max_results=" + maxResults
		} else {
			apiURL += "?max_results=" + maxResults
		}
		log.Printf("Fetching bookmarks with max_results=%s from: %s", maxResults, apiURL)
	}

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

// mergeBookmarks combines API bookmarks with stored bookmarks, removing duplicates
// API bookmarks take precedence (more recent data)
func mergeBookmarks(storedBookmarks *storage.StoredBookmarks, apiBookmarks []models.Bookmark, username string) []models.Bookmark {
	// Start with API bookmarks (most recent)
	bookmarkMap := make(map[string]models.Bookmark)

	// Add API bookmarks first (they have priority)
	for _, bookmark := range apiBookmarks {
		bookmarkMap[bookmark.ID] = bookmark
	}

	// Add stored bookmarks only if they don't exist in API results
	if storedBookmarks != nil && storedBookmarks.Username == username {
		for _, bookmark := range storedBookmarks.Bookmarks {
			if _, exists := bookmarkMap[bookmark.ID]; !exists {
				bookmarkMap[bookmark.ID] = bookmark
			}
		}
	}

	// Convert map back to slice
	result := make([]models.Bookmark, 0, len(bookmarkMap))
	for _, bookmark := range bookmarkMap {
		result = append(result, bookmark)
	}

	log.Printf("Merged bookmarks: API=%d, Stored=%d, Total=%d",
		len(apiBookmarks),
		getStoredBookmarkCount(storedBookmarks, username),
		len(result))

	return result
}

// getStoredBookmarkCount safely gets the count of stored bookmarks for a user
func getStoredBookmarkCount(storedBookmarks *storage.StoredBookmarks, username string) int {
	if storedBookmarks == nil || storedBookmarks.Username != username {
		return 0
	}
	return len(storedBookmarks.Bookmarks)
}
