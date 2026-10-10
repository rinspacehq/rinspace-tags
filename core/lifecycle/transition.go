package lifecycle

import "github.com/rinspacehq/rinspace-tags/core/graph"

type ExternalAction string

const (
	ExternalActionNone    ExternalAction = ""
	ExternalActionArchive ExternalAction = "archive"
	ExternalActionRestore ExternalAction = "restore"
)

type OperationState string

const (
	OperationStateCompleted     OperationState = "completed"
	OperationStateMainCommitted OperationState = "main_committed"
)

// CurrentTag is the transactional state of the Tag being changed.
type CurrentTag struct {
	ID        int64
	Lifecycle graph.Lifecycle
	Version   int64
	Reserved  bool
}

// TargetTag is the transactional state of an optional lifecycle successor.
// A nil target means no target was requested; Exists distinguishes a missing
// requested Tag from an adapter that has not loaded it.
type TargetTag struct {
	ID        int64
	Lifecycle graph.Lifecycle
	Exists    bool
}

// Plan describes only durable domain state. An adapter applies it in the same
// transaction as its product-specific claims, associations, audit, and outbox.
type Plan struct {
	Lifecycle      graph.Lifecycle
	SuccessorTagID *int64
	NewVersion     int64
	ExternalAction ExternalAction
	OperationState OperationState
}

// PlanTransition validates a prepared request against locked current state.
// It does not mutate its inputs or perform I/O.
func PlanTransition(request PreparedRequest, current CurrentTag, target *TargetTag) (Plan, error) {
	if current.ID != request.TagID || current.ID <= 0 {
		return Plan{}, lifecycleError(CodeRequestInvalid)
	}
	if current.Reserved {
		return Plan{}, lifecycleError(CodeReserved, current.ID)
	}
	if current.Version != request.BaseVersion {
		return Plan{}, lifecycleError(CodeVersionConflict, current.ID)
	}

	plan := Plan{NewVersion: current.Version + 1, OperationState: OperationStateCompleted}
	switch request.Operation {
	case OperationArchive:
		if current.Lifecycle != graph.LifecycleActive && current.Lifecycle != graph.LifecycleDeprecated {
			return Plan{}, lifecycleError(CodeConflict, current.ID)
		}
		plan.Lifecycle = graph.LifecycleArchived
		plan.ExternalAction = ExternalActionArchive
		plan.OperationState = OperationStateMainCommitted
	case OperationRestore:
		if current.Lifecycle != graph.LifecycleArchived {
			return Plan{}, lifecycleError(CodeConflict, current.ID)
		}
		plan.Lifecycle = graph.LifecycleActive
		plan.ExternalAction = ExternalActionRestore
		plan.OperationState = OperationStateMainCommitted
	case OperationDeprecate:
		if current.Lifecycle != graph.LifecycleActive {
			return Plan{}, lifecycleError(CodeConflict, current.ID)
		}
		plan.Lifecycle = graph.LifecycleDeprecated
		if request.TargetTagID > 0 {
			successor, err := validateTarget(request.TargetTagID, target)
			if err != nil {
				return Plan{}, err
			}
			plan.SuccessorTagID = &successor
		}
	case OperationMerge:
		if current.Lifecycle != graph.LifecycleActive && current.Lifecycle != graph.LifecycleDeprecated {
			return Plan{}, lifecycleError(CodeConflict, current.ID)
		}
		successor, err := validateTarget(request.TargetTagID, target)
		if err != nil {
			return Plan{}, err
		}
		plan.Lifecycle = graph.LifecycleMerged
		plan.SuccessorTagID = &successor
		plan.ExternalAction = ExternalActionArchive
		plan.OperationState = OperationStateMainCommitted
	default:
		return Plan{}, lifecycleError(CodeRequestInvalid)
	}
	return plan, nil
}

func validateTarget(expectedID int64, target *TargetTag) (int64, error) {
	if target == nil || !target.Exists || target.ID != expectedID {
		return 0, lifecycleError(CodeTargetNotFound, expectedID)
	}
	if target.Lifecycle != graph.LifecycleActive {
		return 0, lifecycleError(CodeTargetConflict, expectedID)
	}
	return expectedID, nil
}
