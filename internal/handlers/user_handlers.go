package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/mepv/go-x-bookmarks/internal/models"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

func UserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	log.Printf("UserHandler: %s", username)
}

func fetchUserInformation(accessToken string, username string) (*models.UserResponse, error) {
	apiURL := fmt.Sprintf("https://api.x.com/2/users/by/username/%s", url.PathEscape(username))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create API request: %w", err)
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
