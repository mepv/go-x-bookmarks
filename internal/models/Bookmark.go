package models

type Bookmark struct {
	EditHistoryTweetIDs []string `json:"edit_history_tweet_ids"`
	Text                string   `json:"text"`
	ID                  string   `json:"id"`
}
