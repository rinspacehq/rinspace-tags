package contracts_test

import (
	"testing"

	"github.com/rinspacehq/rinspace-tags/contracts"
	"github.com/rinspacehq/rinspace-tags/core/graph"
)

func TestNormalizationVectorsV1(t *testing.T) {
	t.Parallel()
	vectors, err := contracts.NormalizationVectorsV1()
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) == 0 {
		t.Fatal("normalization contract contains no vectors")
	}
	for _, vector := range vectors {
		if actual := graph.NormalizeName(vector.Input); actual != vector.Normalized {
			t.Errorf("NormalizeName(%q) = %q, want %q", vector.Input, actual, vector.Normalized)
		}
	}
}
