// Package contracts exposes versioned, implementation-neutral contract
// fixtures shared by the Go core, SDKs, and private consumers.
package contracts

import (
	_ "embed"
	"encoding/json"
)

const NormalizationSchemaV1 = "tag-normalization/v1"

//go:embed tag-normalization-v1.json
var normalizationV1JSON []byte

type NormalizationVector struct {
	Input      string `json:"input"`
	Normalized string `json:"normalized"`
}

type normalizationDocument struct {
	Schema  string                `json:"schema"`
	Vectors []NormalizationVector `json:"vectors"`
}

func NormalizationVectorsV1() ([]NormalizationVector, error) {
	var document normalizationDocument
	if err := json.Unmarshal(normalizationV1JSON, &document); err != nil {
		return nil, err
	}
	vectors := make([]NormalizationVector, len(document.Vectors))
	copy(vectors, document.Vectors)
	return vectors, nil
}
