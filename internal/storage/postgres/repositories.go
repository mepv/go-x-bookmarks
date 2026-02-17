package postgres

import "database/sql"

type Repositories struct {
	Users     *UserRepository
	SyncState *SyncStateRepository
	APIUsage  *APIUsageRepository
}

func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Users:     NewUserRepository(db),
		SyncState: NewSyncStateRepository(db),
		APIUsage:  NewAPIUsageRepository(db),
	}
}
