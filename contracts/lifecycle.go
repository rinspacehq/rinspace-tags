package contracts

import (
	_ "embed"
	"encoding/json"
)

const LifecycleSchemaV1 = "tag-lifecycle/v1"

//go:embed tag-lifecycle-v1.json
var lifecycleV1JSON []byte

type LifecycleRequestVector struct {
	Operation      string `json:"operation"`
	TagID          int64  `json:"tagId"`
	TargetTagID    int64  `json:"targetTagId"`
	BaseVersion    int64  `json:"baseVersion"`
	ActorID        string `json:"actorId"`
	IdempotencyKey string `json:"idempotencyKey"`
	Reason         string `json:"reason"`
	OperationID    string `json:"operationId"`
	RequestHash    string `json:"requestHash"`
}

type LifecycleTransitionVector struct {
	Operation        string `json:"operation"`
	CurrentLifecycle string `json:"currentLifecycle"`
	TargetTagID      int64  `json:"targetTagId"`
	TargetLifecycle  string `json:"targetLifecycle"`
	ResultLifecycle  string `json:"resultLifecycle"`
	ExternalAction   string `json:"externalAction"`
	OperationState   string `json:"operationState"`
}

type LifecycleContractV1 struct {
	Schema      string                      `json:"schema"`
	Requests    []LifecycleRequestVector    `json:"requests"`
	Transitions []LifecycleTransitionVector `json:"transitions"`
}

func LifecycleVectorsV1() (LifecycleContractV1, error) {
	var document LifecycleContractV1
	err := json.Unmarshal(lifecycleV1JSON, &document)
	return document, err
}
