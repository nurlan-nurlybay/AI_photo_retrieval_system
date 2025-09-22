package domain

import "fmt"

type DomainError struct {
	Code string
	Msg  string
}

func (e DomainError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

var (
    ErrMediaNotFound   = DomainError{"MEDIA_NOT_FOUND", "media not found"}
    ErrMediaDeleted    = DomainError{"MEDIA_DELETED", "media is deleted"}
    ErrInvalidDevice   = DomainError{"INVALID_DEVICE", "invalid device"}
    ErrInvalidQuery    = DomainError{"INVALID_QUERY", "invalid query"}
    ErrEmbeddingFailed = DomainError{"EMBEDDING_FAILED", "failed to generate embedding"}
    ErrSearchFailed    = DomainError{"SEARCH_FAILED", "vector search failed"}
)