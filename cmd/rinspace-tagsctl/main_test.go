package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	t.Parallel()
	input := `{"schemaVersion":"rinspace-tags-graph/v1","tags":[{"id":1,"displayName":"Mathematics","normalizedName":"mathematics","parentTagIds":[],"lifecycle":"active","version":1}]}`
	var stdout, stderr bytes.Buffer
	if err := run([]string{"validate"}, strings.NewReader(input), &stdout, &stderr); err != nil {
		t.Fatalf("validate failed: %v, stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"valid": true`) {
		t.Fatalf("unexpected report: %s", stdout.String())
	}
}

func TestValidateCommandRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	input := `{"schemaVersion":"rinspace-tags-graph/v1","tags":[],"secret":"no"}`
	var stdout, stderr bytes.Buffer
	if err := run([]string{"validate"}, strings.NewReader(input), &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected strict decode error, got %v", err)
	}
}
