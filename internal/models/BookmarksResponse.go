package models

type BookmarksResponse struct {
	Data []Bookmark `json:"data"`
	Meta Meta       `json:"meta"`
}
