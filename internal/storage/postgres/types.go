package postgres

import "time"

type UserRecord struct {
	ID                 int64
	XUserID            string
	Username           string
	DisplayName        string
	NotionWorkspaceID  *string
	MetadataFetchedAt  time.Time
	MetadataExpiresAt  time.Time
	LastSuccessfulSync *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type UserSyncState struct {
	UserID             int64
	LastBookmarkCursor *string
	LastSeenTweetID    *string
	LastSyncAt         *time.Time
	LastSuccessfulSync *time.Time
	LastError          *string
	UpdatedAt          time.Time
}

type APIUsageRecord struct {
	ID                 int64
	UserID             int64
	UsageDay           time.Time
	XCallsCount        int32
	XEstimatedCostUnit float64
	NotionWritesCount  int32
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
