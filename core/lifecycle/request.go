package lifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const (
	MaxIdempotencyKeyBytes = 240
	MaxReasonRunes         = 2000
)

type Operation string

const (
	OperationMerge     Operation = "merge"
	OperationDeprecate Operation = "deprecate"
	OperationArchive   Operation = "archive"
	OperationRestore   Operation = "restore"
)

// Request contains the storage-independent intent of one lifecycle change.
// ActorID is opaque: the core neither interprets it nor binds it to an account
// system. Adapters remain responsible for authentication and authorization.
type Request struct {
	Operation      Operation
	TagID          int64
	TargetTagID    int64
	BaseVersion    int64
	ActorID        string
	IdempotencyKey string
	Reason         string
}

// PreparedRequest is a normalized, validated intent with deterministic
// identities suitable for a transactional idempotency record.
type PreparedRequest struct {
	Request
	OperationID string
	RequestHash string
}

// Prepare normalizes textual fields, validates request shape, and derives
// identities. Its request hash preserves Rinspace's original v1 canonical
// JSON field set so existing idempotency records remain replay-compatible.
func Prepare(input Request) (PreparedRequest, error) {
	input.Operation = Operation(strings.TrimSpace(string(input.Operation)))
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.Reason = strings.TrimSpace(input.Reason)

	if input.TagID <= 0 || input.BaseVersion <= 0 || input.ActorID == "" ||
		input.IdempotencyKey == "" || len(input.IdempotencyKey) > MaxIdempotencyKeyBytes ||
		input.Reason == "" || utf8.RuneCountInString(input.Reason) > MaxReasonRunes {
		return PreparedRequest{}, lifecycleError(CodeRequestInvalid)
	}

	switch input.Operation {
	case OperationMerge:
		if input.TargetTagID <= 0 || input.TargetTagID == input.TagID {
			return PreparedRequest{}, lifecycleError(CodeTargetInvalid, input.TargetTagID)
		}
	case OperationDeprecate:
		if input.TargetTagID < 0 || input.TargetTagID == input.TagID {
			return PreparedRequest{}, lifecycleError(CodeTargetInvalid, input.TargetTagID)
		}
	case OperationArchive, OperationRestore:
		if input.TargetTagID != 0 {
			return PreparedRequest{}, lifecycleError(CodeTargetInvalid, input.TargetTagID)
		}
	default:
		return PreparedRequest{}, lifecycleError(CodeRequestInvalid)
	}

	requestJSON, err := json.Marshal(struct {
		BaseVersion   int64     `json:"baseVersion"`
		OperationType Operation `json:"operationType"`
		Reason        string    `json:"reason"`
		TagID         int64     `json:"tagId"`
		TargetTagID   int64     `json:"targetTagId"`
	}{input.BaseVersion, input.Operation, input.Reason, input.TagID, input.TargetTagID})
	if err != nil {
		return PreparedRequest{}, err
	}
	requestDigest := sha256.Sum256(requestJSON)
	idDigest := sha256.Sum256([]byte(input.ActorID + "\x00" + input.IdempotencyKey))

	return PreparedRequest{
		Request:     input,
		OperationID: "tag-life-" + hex.EncodeToString(idDigest[:16]),
		RequestHash: hex.EncodeToString(requestDigest[:]),
	}, nil
}
