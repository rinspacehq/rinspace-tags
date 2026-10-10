// Package lifecycle implements deterministic Tag lifecycle request identity
// and state transitions. Persistence, locking, authorization, and external
// side effects belong to the consuming adapter.
package lifecycle

import "strings"

const (
	CodeRequestInvalid      = "tag_lifecycle_request_invalid"
	CodeTargetInvalid       = "tag_lifecycle_target_invalid"
	CodeConflict            = "tag_lifecycle_conflict"
	CodeTargetNotFound      = "tag_lifecycle_target_not_found"
	CodeTargetConflict      = "tag_lifecycle_target_conflict"
	CodeVersionConflict     = "tag_version_conflict"
	CodeReserved            = "tag_lifecycle_reserved"
	CodeIdempotencyConflict = "tag_idempotency_conflict"
)

// Error exposes a stable machine code and the Tag identities relevant to the
// failure. Message text is intentionally not part of the public contract.
type Error struct {
	Code   string
	TagIDs []int64
}

func (err *Error) Error() string { return err.Code }

func lifecycleError(code string, ids ...int64) *Error {
	return &Error{Code: code, TagIDs: append([]int64(nil), ids...)}
}

// ValidateReplay rejects reuse of an idempotency key for a different request.
func ValidateReplay(expectedHash, actualHash string, tagID int64) error {
	if strings.TrimSpace(expectedHash) == "" || expectedHash != actualHash {
		return lifecycleError(CodeIdempotencyConflict, tagID)
	}
	return nil
}
