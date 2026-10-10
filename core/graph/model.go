// Package graph implements the vendor-neutral relationship kernel used to
// organise Rinspace Tags. It does not depend on Rinspace accounts, HTTP
// sessions, Gitea, Renderer, CloudBase, or a particular storage engine.
package graph

const (
	SchemaV1        = "rinspace-tags-graph/v1"
	MaxParents      = 12
	MaxDisplayRunes = 256
)

type Lifecycle string

const (
	LifecycleActive     Lifecycle = "active"
	LifecycleDeprecated Lifecycle = "deprecated"
	LifecycleArchived   Lifecycle = "archived"
	LifecycleMerged     Lifecycle = "merged"
)

type ReviewState string

const (
	ReviewProposed   ReviewState = "proposed"
	ReviewApproved   ReviewState = "approved"
	ReviewRejected   ReviewState = "rejected"
	ReviewSuperseded ReviewState = "superseded"
)

// Tag carries only facts needed by the relationship kernel. ID is the stable
// identity; names, parents, and lifecycle are governed mutable facts.
type Tag struct {
	ID             int64     `json:"id"`
	DisplayName    string    `json:"displayName"`
	NormalizedName string    `json:"normalizedName"`
	ParentTagIDs   []int64   `json:"parentTagIds"`
	Lifecycle      Lifecycle `json:"lifecycle"`
	SuccessorTagID *int64    `json:"successorTagId,omitempty"`
	Version        int64     `json:"version"`
}

// Alias is a globally resolvable name. It is not a Tag and therefore has no
// parents, Wiki, lifecycle, or independent stable Tag identity.
type Alias struct {
	DisplayName    string `json:"displayName"`
	NormalizedName string `json:"normalizedName"`
	TagID          int64  `json:"tagId"`
	Active         bool   `json:"active"`
}

// Statement represents a governed semantic relation. Classification parents,
// aliases, and lifecycle successors intentionally do not use this type.
type Statement struct {
	ID             string      `json:"id"`
	SubjectTagID   int64       `json:"subjectTagId"`
	PredicateTagID int64       `json:"predicateTagId"`
	ObjectTagID    int64       `json:"objectTagId"`
	ContextTagID   int64       `json:"contextTagId,omitempty"`
	ReviewState    ReviewState `json:"reviewState"`
	Rank           int         `json:"rank"`
}

type Snapshot struct {
	SchemaVersion string           `json:"schemaVersion"`
	Tags          []Tag            `json:"tags"`
	Aliases       []Alias          `json:"aliases,omitempty"`
	Statements    []Statement      `json:"statements,omitempty"`
	Vocabulary    map[string]int64 `json:"vocabulary,omitempty"`
}

type Violation struct {
	Code         string   `json:"code"`
	Message      string   `json:"message"`
	TagIDs       []int64  `json:"tagIds,omitempty"`
	ParentTagIDs []int64  `json:"parentTagIds,omitempty"`
	PathTagIDs   []int64  `json:"pathTagIds,omitempty"`
	StatementIDs []string `json:"statementIds,omitempty"`
}

type Report struct {
	SchemaVersion string      `json:"schemaVersion"`
	Valid         bool        `json:"valid"`
	Violations    []Violation `json:"violations"`
}

type Connection struct {
	TagID        int64 `json:"tagId"`
	MinimumDepth int   `json:"minimumDepth"`
	PathCount    int64 `json:"pathCount"`
}
