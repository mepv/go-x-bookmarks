package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mepv/go-x-bookmarks/internal/config"
	"github.com/mepv/go-x-bookmarks/internal/helpers"
	"github.com/mepv/go-x-bookmarks/internal/models"
	"github.com/mepv/go-x-bookmarks/internal/render"
	"github.com/mepv/go-x-bookmarks/internal/storage"
	storagepostgres "github.com/mepv/go-x-bookmarks/internal/storage/postgres"
)

var app *config.AppConfig
var bookmarkStorage *storage.BookmarkStorage
var metadataStorage *storage.MetadataStorage
var postgresRepos *storagepostgres.Repositories

func NewHandlers(a *config.AppConfig) {
	app = a
	// Initialize bookmark storage to save in data/bookmarks.json
	bookmarkStorage = storage.NewBookmarkStorage("data/bookmarks.json")
	// Initialize metadata storage to save in data/bookmarks-metadata.json
	metadataStorage = storage.NewMetadataStorage("data/bookmarks-metadata.json")

	if app.DB != nil {
		postgresRepos = storagepostgres.NewRepositories(app.DB)
	}
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

	userResp, userRecord, err := resolveUserInformation(r.Context(), accessToken, username)
	if err != nil {
		log.Printf("Error fetching user information: %v", err)
		helpers.ServerError(w, err)
		return
	}

	if usingPostgresOperationalStore() && userRecord != nil {
		if err := postgresRepos.SyncState.MarkSyncAttempt(r.Context(), userRecord.ID, time.Now().UTC(), nil); err != nil {
			log.Printf("failed to mark sync attempt for user_id=%d: %v", userRecord.ID, err)
		}
	}

	// Fetch fresh bookmarks from API
	apiBookmarks, err := fetchBookmarks(accessToken, userResp.Data)
	if err != nil {
		log.Printf("Error fetching bookmarks from API: %v", err)
		if usingPostgresOperationalStore() && userRecord != nil {
			errMsg := err.Error()
			if syncErr := postgresRepos.SyncState.MarkSyncAttempt(r.Context(), userRecord.ID, time.Now().UTC(), &errMsg); syncErr != nil {
				log.Printf("failed to persist sync error for user_id=%d: %v", userRecord.ID, syncErr)
			}
		}
		helpers.ServerError(w, err)
		return
	}

	bookmarks := apiBookmarks
	if !usingPostgresOperationalStore() {
		// Load existing bookmarks from storage
		storedBookmarks, loadErr := bookmarkStorage.LoadBookmarks()
		if loadErr != nil {
			log.Printf("Error loading stored bookmarks: %v", loadErr)
			storedBookmarks = nil // Continue with empty stored bookmarks
		}

		// Merge API bookmarks with stored bookmarks (deduplicate)
		bookmarks = mergeBookmarks(storedBookmarks, apiBookmarks, username)

		// Save merged bookmarks to storage
		saveErr := bookmarkStorage.SaveBookmarks(username, bookmarks)
		if saveErr != nil {
			log.Printf("Error saving merged bookmarks to storage: %v", saveErr)
		} else {
			log.Printf("Saved %d merged bookmarks to storage for user %s", len(bookmarks), username)
		}
	}

	tweetMetadata := make(map[string]models.TweetMetadata)
	if !usingPostgresOperationalStore() {
		// Fetch tweet metadata for bookmarks
		tweetMetadata, err = fetchTweetMetadataForBookmarks(accessToken, username, bookmarks)
		if err != nil {
			log.Printf("Error fetching tweet metadata: %v", err)
			// Continue anyway - we can still show bookmarks without metadata
		}
	}

	if usingPostgresOperationalStore() && userRecord != nil {
		now := time.Now().UTC()
		var lastSeenTweetID *string
		if len(bookmarks) > 0 {
			lastSeenTweetID = &bookmarks[0].ID
		}

		_, syncErr := postgresRepos.SyncState.Upsert(r.Context(), &storagepostgres.UserSyncState{
			UserID:             userRecord.ID,
			LastSeenTweetID:    lastSeenTweetID,
			LastSyncAt:         &now,
			LastSuccessfulSync: &now,
		})
		if syncErr != nil {
			log.Printf("failed to upsert sync state for user_id=%d: %v", userRecord.ID, syncErr)
		}
	}

	data := make(map[string]interface{})
	data["bookmarks"] = bookmarks
	data["username"] = username
	data["metadata"] = tweetMetadata
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

func resolveUserInformation(ctx context.Context, accessToken, username string) (*models.UserResponse, *storagepostgres.UserRecord, error) {
	if postgresRepos == nil {
		userResp, err := fetchUserInformation(accessToken, username)
		return userResp, nil, err
	}

	cachedUser, err := postgresRepos.Users.GetByUsername(ctx, username)
	if err != nil && !errors.Is(err, storagepostgres.ErrNotFound) {
		log.Printf("failed to read user cache for username=%s: %v", username, err)
		return fetchUserInformationAndRefreshCache(ctx, accessToken, username)
	}

	now := time.Now().UTC()
	if err == nil && cachedUser.MetadataExpiresAt.After(now) {
		log.Printf("Cache hit for user=%s (expires_at=%s)", username, cachedUser.MetadataExpiresAt.Format(time.RFC3339))
		return &models.UserResponse{
			Data: models.User{
				ID:       cachedUser.XUserID,
				Name:     cachedUser.DisplayName,
				Username: cachedUser.Username,
			},
		}, cachedUser, nil
	}

	return fetchUserInformationAndRefreshCache(ctx, accessToken, username)
}

func fetchUserInformationAndRefreshCache(ctx context.Context, accessToken, username string) (*models.UserResponse, *storagepostgres.UserRecord, error) {
	userResp, err := fetchUserInformation(accessToken, username)
	if err != nil {
		return nil, nil, err
	}

	if postgresRepos == nil {
		return userResp, nil, nil
	}

	now := time.Now().UTC()
	ttl := getUserCacheTTL()
	updatedRecord, upsertErr := postgresRepos.Users.Upsert(ctx, &storagepostgres.UserRecord{
		XUserID:           userResp.Data.ID,
		Username:          userResp.Data.Username,
		DisplayName:       userResp.Data.Name,
		MetadataFetchedAt: now,
		MetadataExpiresAt: now.Add(ttl),
	})
	if upsertErr != nil {
		log.Printf("failed to upsert user cache for username=%s x_user_id=%s: %v", userResp.Data.Username, userResp.Data.ID, upsertErr)
		return userResp, nil, nil
	}

	return userResp, updatedRecord, nil
}

func getUserCacheTTL() time.Duration {
	const defaultTTLMinutes = 60

	ttlStr := strings.TrimSpace(os.Getenv("USER_CACHE_TTL_MINUTES"))
	if ttlStr == "" {
		return time.Duration(defaultTTLMinutes) * time.Minute
	}

	ttlMinutes, err := strconv.Atoi(ttlStr)
	if err != nil || ttlMinutes <= 0 {
		log.Printf("invalid USER_CACHE_TTL_MINUTES=%q, falling back to %d minutes", ttlStr, defaultTTLMinutes)
		return time.Duration(defaultTTLMinutes) * time.Minute
	}

	return time.Duration(ttlMinutes) * time.Minute
}

func usingPostgresOperationalStore() bool {
	return postgresRepos != nil
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

// fetchTweetMetadataForBookmarks extracts tweet IDs from bookmarks and fetches metadata
func fetchTweetMetadataForBookmarks(accessToken, username string, bookmarks []models.Bookmark) (map[string]models.TweetMetadata, error) {
	if len(bookmarks) == 0 {
		return make(map[string]models.TweetMetadata), nil
	}

	// Extract tweet IDs from bookmarks
	var tweetIDs []string
	for _, bookmark := range bookmarks {
		tweetIDs = append(tweetIDs, bookmark.ID)
	}

	log.Printf("🐦 Extracting metadata for %d tweet IDs", len(tweetIDs))

	// Load existing metadata
	storedMetadata, err := metadataStorage.LoadMetadata()
	if err != nil {
		log.Printf("Error loading stored metadata: %v", err)
		storedMetadata = &models.StoredTweetMetadata{
			Username: username,
			Metadata: make(map[string]models.TweetMetadata),
		}
	}

	// Only fetch metadata for tweets we don't have yet
	newMetadata, err := FetchMissingMetadata(accessToken, tweetIDs, storedMetadata.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch missing metadata: %w", err)
	}

	// Merge new metadata with existing metadata
	allMetadata := storedMetadata.Metadata
	for tweetID, metadata := range newMetadata {
		allMetadata[tweetID] = metadata
	}

	// Save updated metadata to storage
	if len(newMetadata) > 0 {
		err = metadataStorage.SaveMetadata(username, allMetadata)
		if err != nil {
			log.Printf("Error saving updated metadata: %v", err)
		} else {
			log.Printf("✅ Saved metadata for %d tweets (%d new)", len(allMetadata), len(newMetadata))
		}
	}

	// Return metadata only for the current bookmarks
	result := make(map[string]models.TweetMetadata)
	for _, tweetID := range tweetIDs {
		if metadata, exists := allMetadata[tweetID]; exists {
			result[tweetID] = metadata
		}
	}

	return result, nil
}
