package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mepv/go-x-bookmarks/internal/models"
)

const tweetsAPIURL = "https://api.x.com/2/tweets"

// TweetsClient handles fetching tweet metadata from X API
type TweetsClient struct {
	accessToken string
	httpClient  *http.Client
}

func NewTweetsClient(accessToken string) *TweetsClient {
	return &TweetsClient{
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchTweetMetadata retrieves metadata for up to 100 tweet IDs
func (tc *TweetsClient) FetchTweetMetadata(tweetIDs []string) (*models.TweetResponse, error) {
	if len(tweetIDs) == 0 {
		return &models.TweetResponse{Data: []models.TweetData{}}, nil
	}

	if len(tweetIDs) > 100 {
		return nil, fmt.Errorf("too many tweet IDs: %d (max 100)", len(tweetIDs))
	}

	// Build request URL with parameters
	apiURL, err := tc.buildRequestURL(tweetIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build request URL: %w", err)
	}

	log.Printf("🐦 Fetching metadata for %d tweets from: %s", len(tweetIDs), apiURL)

	// Create request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+tc.accessToken)
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("⚠️ Error closing response body: %v", err)
		}
	}(resp.Body)

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("🐦 Tweet API response - Status: %d", resp.StatusCode)

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Tweet API error response: %s", string(body))
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tweetResp models.TweetResponse
	if err := json.Unmarshal(body, &tweetResp); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	log.Printf("✅ Successfully fetched metadata for %d tweets", len(tweetResp.Data))
	if len(tweetResp.Errors) > 0 {
		log.Printf("⚠️ API returned %d errors", len(tweetResp.Errors))
		for _, apiErr := range tweetResp.Errors {
			log.Printf("⚠️ API Error: %s - %s", apiErr.Type, apiErr.Detail)
		}
	}

	return &tweetResp, nil
}

// buildRequestURL constructs the API URL with all required parameters
func (tc *TweetsClient) buildRequestURL(tweetIDs []string) (string, error) {
	baseURL, err := url.Parse(tweetsAPIURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	
	// Required: comma-separated tweet IDs
	params.Add("ids", strings.Join(tweetIDs, ","))
	
	// Tweet fields we want
	tweetFields := []string{
		"id",
		"text", 
		"author_id",
		"created_at",
		"public_metrics",
		"lang",
		"source",
		"in_reply_to_user_id",
	}
	params.Add("tweet.fields", strings.Join(tweetFields, ","))
	
	// Expansions to get additional data
	expansions := []string{
		"author_id",
	}
	params.Add("expansions", strings.Join(expansions, ","))
	
	// User fields for author information  
	userFields := []string{
		"id",
		"name",
		"username",
		"profile_image_url",
		"verified",
		"public_metrics",
	}
	params.Add("user.fields", strings.Join(userFields, ","))

	baseURL.RawQuery = params.Encode()
	return baseURL.String(), nil
}

// ConvertToTweetMetadata converts API response to our internal metadata format
func ConvertToTweetMetadata(tweetResp *models.TweetResponse) (map[string]models.TweetMetadata, error) {
	if tweetResp == nil {
		return make(map[string]models.TweetMetadata), nil
	}

	result := make(map[string]models.TweetMetadata)
	
	// Create a map of user ID to user for easy lookup
	userMap := make(map[string]models.User)
	if tweetResp.Includes != nil {
		for _, user := range tweetResp.Includes.Users {
			userMap[user.ID] = user
		}
	}

	// Convert each tweet
	for _, tweetData := range tweetResp.Data {
		metadata := models.TweetMetadata{
			ID:            tweetData.ID,
			Text:          tweetData.Text,
			AuthorID:      tweetData.AuthorID,
			PublicMetrics: tweetData.PublicMetrics,
			Lang:          tweetData.Lang,
			Source:        tweetData.Source,
			ReplyToID:     tweetData.ReplyToID,
		}

		// Parse created_at timestamp
		if tweetData.CreatedAt != "" {
			if createdAt, err := time.Parse(time.RFC3339, tweetData.CreatedAt); err == nil {
				metadata.CreatedAt = createdAt
			} else {
				log.Printf("⚠️ Failed to parse created_at for tweet %s: %v", tweetData.ID, err)
			}
		}

		// Add author information if available
		if author, exists := userMap[tweetData.AuthorID]; exists {
			metadata.Author = &author
		}

		result[tweetData.ID] = metadata
	}

	return result, nil
}

// FetchMissingMetadata is a helper that only fetches metadata for tweets that don't have it yet
func FetchMissingMetadata(accessToken string, allTweetIDs []string, existingMetadata map[string]models.TweetMetadata) (map[string]models.TweetMetadata, error) {
	// Find missing tweet IDs
	var missingIDs []string
	for _, tweetID := range allTweetIDs {
		if _, exists := existingMetadata[tweetID]; !exists {
			missingIDs = append(missingIDs, tweetID)
		}
	}

	if len(missingIDs) == 0 {
		log.Printf("✅ All tweet metadata already exists, skipping API call")
		return make(map[string]models.TweetMetadata), nil
	}

	log.Printf("🐦 Fetching metadata for %d missing tweets out of %d total", len(missingIDs), len(allTweetIDs))

	// Batch API calls if we have more than 100 missing IDs
	allNewMetadata := make(map[string]models.TweetMetadata)
	
	for i := 0; i < len(missingIDs); i += 100 {
		end := i + 100
		if end > len(missingIDs) {
			end = len(missingIDs)
		}
		
		batch := missingIDs[i:end]
		log.Printf("🐦 Processing batch %d-%d (%d tweets)", i+1, end, len(batch))
		
		client := NewTweetsClient(accessToken)
		tweetResp, err := client.FetchTweetMetadata(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch batch %d-%d: %w", i+1, end, err)
		}
		
		batchMetadata, err := ConvertToTweetMetadata(tweetResp)
		if err != nil {
			return nil, fmt.Errorf("failed to convert batch %d-%d: %w", i+1, end, err)
		}
		
		// Merge batch results
		for tweetID, metadata := range batchMetadata {
			allNewMetadata[tweetID] = metadata
		}
		
		// Rate limiting: wait between batches (only 1 req per 15 min on free tier)
		if end < len(missingIDs) {
			log.Printf("⏳ Waiting 15 minutes for rate limit before next batch...")
			time.Sleep(15 * time.Minute)
		}
	}

	return allNewMetadata, nil
}