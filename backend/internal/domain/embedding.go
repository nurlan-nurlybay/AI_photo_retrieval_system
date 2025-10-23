package domain

import (
	"time"
)

type Embedding struct {
	MediaID   int64
	UserID    int64
	Model     string
	VecBytes  []byte
	Status    string
	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
}
