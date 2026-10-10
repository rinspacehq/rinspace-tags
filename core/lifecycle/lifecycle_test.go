package lifecycle

import (
	"errors"
	"strings"
	"testing"

	"github.com/rinspacehq/rinspace-tags/core/graph"
)

func TestPreparePreservesV1RequestIdentity(t *testing.T) {
	prepared, err := Prepare(Request{
		Operation: OperationMerge, TagID: 17, TargetTagID: 23, BaseVersion: 4,
		ActorID: " curator ", IdempotencyKey: " merge-17 ", Reason: " equivalent identity ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.ActorID != "curator" || prepared.IdempotencyKey != "merge-17" || prepared.Reason != "equivalent identity" {
		t.Fatalf("request was not normalized: %#v", prepared.Request)
	}
	if prepared.OperationID != "tag-life-c32cba309c00fb545e0ffbe8ad4a1391" {
		t.Fatalf("operation identity changed: %s", prepared.OperationID)
	}
	if prepared.RequestHash != "0bf900d6eefbc3eb9ff7ced89015b7e183225e821e1df72ccb90a987ba6d7e36" {
		t.Fatalf("request identity changed: %s", prepared.RequestHash)
	}
}

func TestPrepareRejectsAmbiguousTargetsAndBounds(t *testing.T) {
	base := Request{Operation: OperationArchive, TagID: 7, BaseVersion: 1, ActorID: "actor", IdempotencyKey: "key", Reason: "reason"}
	tests := []struct {
		name string
		edit func(*Request)
		code string
	}{
		{"unknown operation", func(input *Request) { input.Operation = "remove" }, CodeRequestInvalid},
		{"merge target required", func(input *Request) { input.Operation = OperationMerge }, CodeTargetInvalid},
		{"self successor", func(input *Request) { input.Operation = OperationDeprecate; input.TargetTagID = input.TagID }, CodeTargetInvalid},
		{"archive target forbidden", func(input *Request) { input.TargetTagID = 9 }, CodeTargetInvalid},
		{"long key", func(input *Request) { input.IdempotencyKey = strings.Repeat("x", MaxIdempotencyKeyBytes+1) }, CodeRequestInvalid},
		{"long reason", func(input *Request) { input.Reason = strings.Repeat("界", MaxReasonRunes+1) }, CodeRequestInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := base
			test.edit(&input)
			_, err := Prepare(input)
			requireCode(t, err, test.code)
		})
	}
}

func TestPlanTransitionMatrix(t *testing.T) {
	tests := []struct {
		name     string
		op       Operation
		from     graph.Lifecycle
		targetID int64
		target   *TargetTag
		want     graph.Lifecycle
		action   ExternalAction
		state    OperationState
	}{
		{"archive active", OperationArchive, graph.LifecycleActive, 0, nil, graph.LifecycleArchived, ExternalActionArchive, OperationStateMainCommitted},
		{"archive deprecated", OperationArchive, graph.LifecycleDeprecated, 0, nil, graph.LifecycleArchived, ExternalActionArchive, OperationStateMainCommitted},
		{"restore archived", OperationRestore, graph.LifecycleArchived, 0, nil, graph.LifecycleActive, ExternalActionRestore, OperationStateMainCommitted},
		{"deprecate", OperationDeprecate, graph.LifecycleActive, 0, nil, graph.LifecycleDeprecated, ExternalActionNone, OperationStateCompleted},
		{"deprecate with successor", OperationDeprecate, graph.LifecycleActive, 12, &TargetTag{ID: 12, Lifecycle: graph.LifecycleActive, Exists: true}, graph.LifecycleDeprecated, ExternalActionNone, OperationStateCompleted},
		{"merge active", OperationMerge, graph.LifecycleActive, 12, &TargetTag{ID: 12, Lifecycle: graph.LifecycleActive, Exists: true}, graph.LifecycleMerged, ExternalActionArchive, OperationStateMainCommitted},
		{"merge deprecated", OperationMerge, graph.LifecycleDeprecated, 12, &TargetTag{ID: 12, Lifecycle: graph.LifecycleActive, Exists: true}, graph.LifecycleMerged, ExternalActionArchive, OperationStateMainCommitted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepared := mustPrepare(t, Request{Operation: test.op, TagID: 7, TargetTagID: test.targetID, BaseVersion: 3, ActorID: "actor", IdempotencyKey: test.name, Reason: "governance"})
			plan, err := PlanTransition(prepared, CurrentTag{ID: 7, Lifecycle: test.from, Version: 3}, test.target)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Lifecycle != test.want || plan.ExternalAction != test.action || plan.OperationState != test.state || plan.NewVersion != 4 {
				t.Fatalf("unexpected plan: %#v", plan)
			}
			if (test.targetID > 0) != (plan.SuccessorTagID != nil) {
				t.Fatalf("unexpected successor: %#v", plan.SuccessorTagID)
			}
		})
	}
}

func TestPlanTransitionRejectsConflicts(t *testing.T) {
	archive := mustPrepare(t, Request{Operation: OperationArchive, TagID: 7, BaseVersion: 3, ActorID: "actor", IdempotencyKey: "archive", Reason: "reason"})
	merge := mustPrepare(t, Request{Operation: OperationMerge, TagID: 7, TargetTagID: 12, BaseVersion: 3, ActorID: "actor", IdempotencyKey: "merge", Reason: "reason"})
	tests := []struct {
		name    string
		request PreparedRequest
		current CurrentTag
		target  *TargetTag
		code    string
	}{
		{"reserved", archive, CurrentTag{ID: 7, Lifecycle: graph.LifecycleActive, Version: 3, Reserved: true}, nil, CodeReserved},
		{"version", archive, CurrentTag{ID: 7, Lifecycle: graph.LifecycleActive, Version: 4}, nil, CodeVersionConflict},
		{"transition", archive, CurrentTag{ID: 7, Lifecycle: graph.LifecycleMerged, Version: 3}, nil, CodeConflict},
		{"missing target", merge, CurrentTag{ID: 7, Lifecycle: graph.LifecycleActive, Version: 3}, nil, CodeTargetNotFound},
		{"inactive target", merge, CurrentTag{ID: 7, Lifecycle: graph.LifecycleActive, Version: 3}, &TargetTag{ID: 12, Lifecycle: graph.LifecycleArchived, Exists: true}, CodeTargetConflict},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := PlanTransition(test.request, test.current, test.target)
			requireCode(t, err, test.code)
		})
	}
}

func TestValidateReplay(t *testing.T) {
	if err := ValidateReplay("same", "same", 7); err != nil {
		t.Fatal(err)
	}
	requireCode(t, ValidateReplay("first", "second", 7), CodeIdempotencyConflict)
}

func mustPrepare(t *testing.T, request Request) PreparedRequest {
	t.Helper()
	prepared, err := Prepare(request)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var lifecycleErr *Error
	if !errors.As(err, &lifecycleErr) || lifecycleErr.Code != code {
		t.Fatalf("error=%v, want lifecycle code %q", err, code)
	}
}
