package contracts_test

import (
	"testing"

	"github.com/rinspacehq/rinspace-tags/contracts"
	"github.com/rinspacehq/rinspace-tags/core/graph"
	"github.com/rinspacehq/rinspace-tags/core/lifecycle"
)

func TestLifecycleVectorsV1(t *testing.T) {
	t.Parallel()
	document, err := contracts.LifecycleVectorsV1()
	if err != nil {
		t.Fatal(err)
	}
	if document.Schema != contracts.LifecycleSchemaV1 || len(document.Requests) == 0 || len(document.Transitions) == 0 {
		t.Fatalf("incomplete lifecycle contract: %#v", document)
	}
	for _, vector := range document.Requests {
		prepared, err := lifecycle.Prepare(lifecycle.Request{
			Operation: lifecycle.Operation(vector.Operation), TagID: vector.TagID,
			TargetTagID: vector.TargetTagID, BaseVersion: vector.BaseVersion,
			ActorID: vector.ActorID, IdempotencyKey: vector.IdempotencyKey, Reason: vector.Reason,
		})
		if err != nil {
			t.Fatal(err)
		}
		if prepared.OperationID != vector.OperationID || prepared.RequestHash != vector.RequestHash {
			t.Errorf("request %s identity=(%s,%s), want=(%s,%s)", vector.IdempotencyKey, prepared.OperationID, prepared.RequestHash, vector.OperationID, vector.RequestHash)
		}
	}
	for index, vector := range document.Transitions {
		request := lifecycle.Request{
			Operation: lifecycle.Operation(vector.Operation), TagID: 17,
			TargetTagID: vector.TargetTagID, BaseVersion: 4,
			ActorID: "contract", IdempotencyKey: vector.Operation + "-contract-" + string(rune('a'+index)), Reason: "contract vector",
		}
		prepared, err := lifecycle.Prepare(request)
		if err != nil {
			t.Fatal(err)
		}
		var target *lifecycle.TargetTag
		if vector.TargetTagID > 0 {
			target = &lifecycle.TargetTag{ID: vector.TargetTagID, Lifecycle: graph.Lifecycle(vector.TargetLifecycle), Exists: true}
		}
		plan, err := lifecycle.PlanTransition(prepared, lifecycle.CurrentTag{ID: 17, Lifecycle: graph.Lifecycle(vector.CurrentLifecycle), Version: 4}, target)
		if err != nil {
			t.Fatal(err)
		}
		if string(plan.Lifecycle) != vector.ResultLifecycle || string(plan.ExternalAction) != vector.ExternalAction || string(plan.OperationState) != vector.OperationState {
			t.Errorf("transition %d plan=%#v", index, plan)
		}
	}
}
