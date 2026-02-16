package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mepv/go-x-bookmarks/internal/models"
)

type MetadataStorage struct {
	filePath string
}

func NewMetadataStorage(filePath string) *MetadataStorage {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Log error but continue - we'll handle file errors when reading/writing
		fmt.Printf("Warning: Could not create directory %s: %v\n", dir, err)
	}

	return &MetadataStorage{
		filePath: filePath,
	}
}

func (ms *MetadataStorage) SaveMetadata(username string, metadata map[string]models.TweetMetadata) error {
	storedMetadata := models.StoredTweetMetadata{
		LastUpdated: time.Now(),
		Username:    username,
		Metadata:    metadata,
	}

	jsonData, err := json.MarshalIndent(storedMetadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}

	err = os.WriteFile(ms.filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metadata to file: %w", err)
	}

	return nil
}

func (ms *MetadataStorage) LoadMetadata() (*models.StoredTweetMetadata, error) {
	file, err := os.Open(ms.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty metadata
			return &models.StoredTweetMetadata{
				LastUpdated: time.Time{},
				Username:    "",
				Metadata:    make(map[string]models.TweetMetadata),
			}, nil
		}
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			_ = fmt.Errorf("failed to close metada file: %w", err)
		}
	}(file)

	jsonData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var storedMetadata models.StoredTweetMetadata
	err = json.Unmarshal(jsonData, &storedMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata JSON: %w", err)
	}

	// Initialize metadata map if nil
	if storedMetadata.Metadata == nil {
		storedMetadata.Metadata = make(map[string]models.TweetMetadata)
	}

	return &storedMetadata, nil
}

func (ms *MetadataStorage) FileExists() bool {
	_, err := os.Stat(ms.filePath)
	return !os.IsNotExist(err)
}

func (ms *MetadataStorage) GetLastUpdated() (time.Time, error) {
	storedMetadata, err := ms.LoadMetadata()
	if err != nil {
		return time.Time{}, err
	}
	return storedMetadata.LastUpdated, nil
}

// AddOrUpdateMetadata adds new metadata or updates existing ones
func (ms *MetadataStorage) AddOrUpdateMetadata(username string, newMetadata map[string]models.TweetMetadata) error {
	// Load existing metadata
	storedMetadata, err := ms.LoadMetadata()
	if err != nil {
		return fmt.Errorf("failed to load existing metadata: %w", err)
	}

	// If username doesn't match, start fresh
	if storedMetadata.Username != username {
		storedMetadata.Username = username
		storedMetadata.Metadata = make(map[string]models.TweetMetadata)
	}

	// Merge new metadata with existing
	for tweetID, metadata := range newMetadata {
		storedMetadata.Metadata[tweetID] = metadata
	}

	// Save the updated metadata
	return ms.SaveMetadata(username, storedMetadata.Metadata)
}

// GetMetadataForTweets returns metadata for specific tweet IDs
func (ms *MetadataStorage) GetMetadataForTweets(tweetIDs []string) (map[string]models.TweetMetadata, error) {
	storedMetadata, err := ms.LoadMetadata()
	if err != nil {
		return nil, err
	}

	result := make(map[string]models.TweetMetadata)
	for _, tweetID := range tweetIDs {
		if metadata, exists := storedMetadata.Metadata[tweetID]; exists {
			result[tweetID] = metadata
		}
	}

	return result, nil
}

// GetMissingMetadataIDs returns tweet IDs that don't have metadata yet
func (ms *MetadataStorage) GetMissingMetadataIDs(tweetIDs []string) ([]string, error) {
	storedMetadata, err := ms.LoadMetadata()
	if err != nil {
		return nil, err
	}

	var missing []string
	for _, tweetID := range tweetIDs {
		if _, exists := storedMetadata.Metadata[tweetID]; !exists {
			missing = append(missing, tweetID)
		}
	}

	return missing, nil
}