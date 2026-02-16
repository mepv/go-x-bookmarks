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

type BookmarkStorage struct {
	filePath string
}

type StoredBookmarks struct {
	LastUpdated time.Time         `json:"last_updated"`
	Username    string            `json:"username"`
	Bookmarks   []models.Bookmark `json:"bookmarks"`
}

func NewBookmarkStorage(filePath string) *BookmarkStorage {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Log error but continue - we'll handle file errors when reading/writing
		fmt.Printf("Warning: Could not create directory %s: %v\n", dir, err)
	}

	return &BookmarkStorage{
		filePath: filePath,
	}
}

func (bs *BookmarkStorage) SaveBookmarks(username string, bookmarks []models.Bookmark) error {
	storedBookmarks := StoredBookmarks{
		LastUpdated: time.Now(),
		Username:    username,
		Bookmarks:   bookmarks,
	}

	jsonData, err := json.MarshalIndent(storedBookmarks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bookmarks to JSON: %w", err)
	}

	err = os.WriteFile(bs.filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write bookmarks to file: %w", err)
	}

	return nil
}

func (bs *BookmarkStorage) LoadBookmarks() (*StoredBookmarks, error) {
	file, err := os.Open(bs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty bookmarks
			return &StoredBookmarks{
				LastUpdated: time.Time{},
				Username:    "",
				Bookmarks:   []models.Bookmark{},
			}, nil
		}
		return nil, fmt.Errorf("failed to open bookmarks file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			_ = fmt.Errorf("failed to close file resource: %w", err)
		}
	}(file)

	jsonData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read bookmarks file: %w", err)
	}

	var storedBookmarks StoredBookmarks
	err = json.Unmarshal(jsonData, &storedBookmarks)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bookmarks JSON: %w", err)
	}

	return &storedBookmarks, nil
}

func (bs *BookmarkStorage) FileExists() bool {
	_, err := os.Stat(bs.filePath)
	return !os.IsNotExist(err)
}

func (bs *BookmarkStorage) GetLastUpdated() (time.Time, error) {
	storedBookmarks, err := bs.LoadBookmarks()
	if err != nil {
		return time.Time{}, err
	}
	return storedBookmarks.LastUpdated, nil
}
