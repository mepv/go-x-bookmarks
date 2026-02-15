package models

import "time"

// TweetMetadata represents rich metadata for a tweet/post
type TweetMetadata struct {
	ID          string       `json:"id"`
	Text        string       `json:"text"`
	AuthorID    string       `json:"author_id"`
	CreatedAt   time.Time    `json:"created_at"`
	PublicMetrics PublicMetrics `json:"public_metrics,omitempty"`
	Author      *User        `json:"author,omitempty"`
	// Additional fields that might be useful
	Lang        string       `json:"lang,omitempty"`
	Source      string       `json:"source,omitempty"`
	ReplyToID   string       `json:"in_reply_to_user_id,omitempty"`
}

// PublicMetrics represents engagement metrics for a tweet
type PublicMetrics struct {
	RetweetCount int `json:"retweet_count"`
	LikeCount    int `json:"like_count"`
	ReplyCount   int `json:"reply_count"`
	QuoteCount   int `json:"quote_count"`
}

// TweetResponse represents the API response from GET /2/tweets
type TweetResponse struct {
	Data     []TweetData `json:"data"`
	Includes *Includes   `json:"includes,omitempty"`
	Errors   []APIError  `json:"errors,omitempty"`
}

// TweetData represents individual tweet data from API response
type TweetData struct {
	ID            string        `json:"id"`
	Text          string        `json:"text"`
	AuthorID      string        `json:"author_id"`
	CreatedAt     string        `json:"created_at"` // API returns as string
	PublicMetrics PublicMetrics `json:"public_metrics,omitempty"`
	Lang          string        `json:"lang,omitempty"`
	Source        string        `json:"source,omitempty"`
	ReplyToID     string        `json:"in_reply_to_user_id,omitempty"`
}

// Includes represents expanded data from API response
type Includes struct {
	Users []User `json:"users,omitempty"`
	Media []Media `json:"media,omitempty"`
}

// Media represents media attachments
type Media struct {
	MediaKey string `json:"media_key"`
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
	PreviewImageURL string `json:"preview_image_url,omitempty"`
}

// APIError represents errors in API responses
type APIError struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
	Title  string `json:"title"`
	Value  string `json:"value,omitempty"`
}

// StoredTweetMetadata represents the structure for storing metadata
type StoredTweetMetadata struct {
	LastUpdated time.Time                `json:"last_updated"`
	Username    string                   `json:"username"`
	Metadata    map[string]TweetMetadata `json:"metadata"` // Map of tweet ID to metadata
}