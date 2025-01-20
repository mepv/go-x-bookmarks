package models

type Bookmark struct {
	EditHistoryTweetIDs []string `json:"edit_history_tweet_ids"`
	Text                string   `json:"text"`
	ID                  string   `json:"id"`
}

type BookmarksResponse struct {
	Data []Bookmark `json:"data"`
	Meta Meta       `json:"meta"`
}

type Meta struct {
	ResultCount int `json:"result_count"`
	//todo: add next_token and previous_token
}
