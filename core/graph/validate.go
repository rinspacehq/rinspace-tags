package graph

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const ReportSchemaV1 = "rinspace-tags-validation-report/v1"

// Validate returns every independently detectable violation in deterministic
// order. It never mutates the supplied snapshot.
func Validate(snapshot Snapshot) Report {
	validator := snapshotValidator{
		snapshot:   snapshot,
		tags:       make(map[int64]Tag, len(snapshot.Tags)),
		parents:    make(map[int64][]int64, len(snapshot.Tags)),
		violations: []Violation{},
	}
	validator.validate()
	sort.SliceStable(validator.violations, func(i, j int) bool {
		left, right := validator.violations[i], validator.violations[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		return violationKey(left) < violationKey(right)
	})
	return Report{
		SchemaVersion: ReportSchemaV1,
		Valid:         len(validator.violations) == 0,
		Violations:    validator.violations,
	}
}

type snapshotValidator struct {
	snapshot   Snapshot
	tags       map[int64]Tag
	parents    map[int64][]int64
	violations []Violation
}

func (validator *snapshotValidator) validate() {
	if validator.snapshot.SchemaVersion != SchemaV1 {
		validator.add("tag_graph_schema_invalid", "schemaVersion must be "+SchemaV1, nil, nil, nil, nil)
	}
	validator.validateTags()
	validator.validateParents()
	validator.validateParentCycles()
	validator.validateNameContexts()
	validator.validateSuccessors()
	validator.validateAliases()
	validator.validateStatements()
}

func (validator *snapshotValidator) validateTags() {
	for _, tag := range validator.snapshot.Tags {
		if tag.ID <= 0 {
			validator.add("tag_id_invalid", "Tag ID must be a positive integer", []int64{tag.ID}, nil, nil, nil)
			continue
		}
		if _, exists := validator.tags[tag.ID]; exists {
			validator.add("tag_id_duplicate", "Tag ID must be unique", []int64{tag.ID}, nil, nil, nil)
			continue
		}
		validator.tags[tag.ID] = tag
		if strings.TrimSpace(tag.DisplayName) == "" || utf8.RuneCountInString(tag.DisplayName) > MaxDisplayRunes {
			validator.add("tag_name_invalid", "displayName must contain 1 to 256 Unicode code points", []int64{tag.ID}, nil, nil, nil)
		}
		normalized := NormalizeName(tag.DisplayName)
		if normalized == "" || tag.NormalizedName != normalized {
			validator.add("tag_normalized_name_mismatch", "normalizedName does not match the versioned canonical transform", []int64{tag.ID}, nil, nil, nil)
		}
		if tag.Version <= 0 {
			validator.add("tag_version_invalid", "Tag governance version must be positive", []int64{tag.ID}, nil, nil, nil)
		}
		if !validLifecycle(tag.Lifecycle) {
			validator.add("tag_lifecycle_invalid", "Tag lifecycle is not recognised", []int64{tag.ID}, nil, nil, nil)
		}
	}
}

func (validator *snapshotValidator) validateParents() {
	for id, tag := range validator.tags {
		seen := make(map[int64]struct{}, len(tag.ParentTagIDs))
		canonical := true
		last := int64(0)
		for index, parentID := range tag.ParentTagIDs {
			if parentID <= 0 {
				validator.add("tag_parent_invalid", "Parent Tag ID must be positive", []int64{id}, []int64{parentID}, nil, nil)
				continue
			}
			if _, duplicate := seen[parentID]; duplicate {
				validator.add("tag_parent_duplicate", "A direct parent may appear only once", []int64{id}, []int64{parentID}, nil, nil)
				canonical = false
				continue
			}
			seen[parentID] = struct{}{}
			if index > 0 && parentID <= last {
				canonical = false
			}
			last = parentID
			parent, exists := validator.tags[parentID]
			if !exists {
				validator.add("tag_parent_not_found", "Direct parent does not exist", []int64{id}, []int64{parentID}, nil, nil)
				continue
			}
			if parentID == id {
				validator.add("tag_parent_cycle", "A Tag cannot be its own parent", []int64{id}, []int64{parentID}, []int64{id, id}, nil)
			}
			if tag.Lifecycle == LifecycleActive && parent.Lifecycle != LifecycleActive {
				validator.add("tag_parent_lifecycle_conflict", "An active Tag requires active direct parents", []int64{id, parentID}, []int64{parentID}, nil, nil)
			}
		}
		if len(tag.ParentTagIDs) > MaxParents {
			validator.add("tag_parent_limit_exceeded", "A Tag may have at most 12 direct parents", []int64{id}, append([]int64(nil), tag.ParentTagIDs...), nil, nil)
		}
		if !canonical {
			validator.add("tag_parent_ids_not_canonical", "parentTagIds must be unique and sorted in ascending order", []int64{id}, append([]int64(nil), tag.ParentTagIDs...), nil, nil)
		}
		validator.parents[id] = sortedUniquePositive(tag.ParentTagIDs)
	}
}

func (validator *snapshotValidator) validateParentCycles() {
	state := make(map[int64]uint8, len(validator.tags))
	stack := make([]int64, 0, len(validator.tags))
	index := make(map[int64]int, len(validator.tags))
	seenCycles := make(map[string]struct{})
	var visit func(int64)
	visit = func(id int64) {
		state[id] = 1
		index[id] = len(stack)
		stack = append(stack, id)
		for _, parentID := range validator.parents[id] {
			if _, exists := validator.tags[parentID]; !exists || parentID == id {
				continue
			}
			switch state[parentID] {
			case 0:
				visit(parentID)
			case 1:
				start := index[parentID]
				path := append([]int64(nil), stack[start:]...)
				path = append(path, parentID)
				key := int64List(path)
				if _, duplicate := seenCycles[key]; !duplicate {
					seenCycles[key] = struct{}{}
					validator.add("tag_parent_cycle", "Direct parent relationships must form a directed acyclic graph", append([]int64(nil), path...), nil, path, nil)
				}
			}
		}
		stack = stack[:len(stack)-1]
		delete(index, id)
		state[id] = 2
	}
	for _, id := range sortedTagIDs(validator.tags) {
		if state[id] == 0 {
			visit(id)
		}
	}
}

func (validator *snapshotValidator) validateNameContexts() {
	groups := make(map[string][]Tag)
	for _, tag := range validator.tags {
		if tag.Lifecycle == LifecycleActive && tag.NormalizedName != "" {
			groups[tag.NormalizedName] = append(groups[tag.NormalizedName], tag)
		}
	}
	for _, group := range groups {
		sort.Slice(group, func(i, j int) bool { return group[i].ID < group[j].ID })
		for left := 0; left < len(group); left++ {
			for right := left + 1; right < len(group); right++ {
				parents := overlappingContext(group[left].ParentTagIDs, group[right].ParentTagIDs)
				if parents != nil {
					validator.add("tag_name_context_conflict", "Active Tags with the same normalized name require disjoint direct-parent contexts", []int64{group[left].ID, group[right].ID}, parents, nil, nil)
				}
			}
		}
	}
}

func (validator *snapshotValidator) validateSuccessors() {
	for id, tag := range validator.tags {
		hasSuccessor := tag.SuccessorTagID != nil && *tag.SuccessorTagID > 0
		if tag.Lifecycle == LifecycleMerged && !hasSuccessor {
			validator.add("tag_successor_required", "A merged Tag requires a successor", []int64{id}, nil, nil, nil)
			continue
		}
		if (tag.Lifecycle == LifecycleActive || tag.Lifecycle == LifecycleArchived) && tag.SuccessorTagID != nil {
			validator.add("tag_successor_forbidden", "Active and archived Tags cannot carry successors", []int64{id}, nil, nil, nil)
		}
		if !hasSuccessor {
			continue
		}
		successorID := *tag.SuccessorTagID
		successor, exists := validator.tags[successorID]
		if !exists {
			validator.add("tag_successor_not_found", "Successor Tag does not exist", []int64{id, successorID}, nil, nil, nil)
			continue
		}
		if successorID == id {
			validator.add("tag_successor_cycle", "A Tag cannot succeed itself", []int64{id}, nil, []int64{id, id}, nil)
		}
		if tag.Lifecycle == LifecycleMerged && successor.Lifecycle != LifecycleActive {
			validator.add("tag_successor_lifecycle_conflict", "A merged Tag must point to an active successor", []int64{id, successorID}, nil, nil, nil)
		}
	}
	validator.validateSuccessorCycles()
}

func (validator *snapshotValidator) validateSuccessorCycles() {
	for _, start := range sortedTagIDs(validator.tags) {
		positions := map[int64]int{}
		path := []int64{}
		current := start
		for {
			if position, exists := positions[current]; exists {
				cycle := append([]int64(nil), path[position:]...)
				cycle = append(cycle, current)
				if len(cycle) > 2 || cycle[0] != cycle[1] {
					validator.add("tag_successor_cycle", "Lifecycle successor relationships must not cycle", append([]int64(nil), cycle...), nil, cycle, nil)
				}
				break
			}
			positions[current] = len(path)
			path = append(path, current)
			tag, exists := validator.tags[current]
			if !exists || tag.SuccessorTagID == nil || *tag.SuccessorTagID <= 0 {
				break
			}
			current = *tag.SuccessorTagID
		}
	}
}

func (validator *snapshotValidator) validateAliases() {
	active := make(map[string]Alias)
	for _, alias := range validator.snapshot.Aliases {
		normalized := NormalizeName(alias.DisplayName)
		if normalized == "" || normalized != alias.NormalizedName {
			validator.add("tag_alias_normalized_name_mismatch", "Alias normalizedName does not match the canonical transform", []int64{alias.TagID}, nil, nil, nil)
		}
		tag, exists := validator.tags[alias.TagID]
		if !exists {
			validator.add("tag_alias_target_not_found", "Alias target Tag does not exist", []int64{alias.TagID}, nil, nil, nil)
			continue
		}
		if !alias.Active {
			continue
		}
		if tag.Lifecycle != LifecycleActive {
			validator.add("tag_alias_target_lifecycle_conflict", "An active alias must resolve to an active Tag", []int64{alias.TagID}, nil, nil, nil)
		}
		if prior, duplicate := active[alias.NormalizedName]; duplicate {
			validator.add("tag_alias_conflict", "An active normalized alias may resolve to only one Tag", []int64{prior.TagID, alias.TagID}, nil, nil, nil)
		} else {
			active[alias.NormalizedName] = alias
		}
	}
}

func (validator *snapshotValidator) validateStatements() {
	requiresID := validator.snapshot.Vocabulary["requires"]
	if requiresID != 0 {
		tag, exists := validator.tags[requiresID]
		if !exists || tag.Lifecycle != LifecycleActive {
			validator.add("tag_vocabulary_binding_invalid", "The requires vocabulary binding must name an active Tag", []int64{requiresID}, nil, nil, nil)
		}
	}
	identities := make(map[string]string)
	requiresEdges := make(map[int64][]int64)
	requiresStatement := make(map[string][]string)
	for _, statement := range validator.snapshot.Statements {
		if strings.TrimSpace(statement.ID) == "" {
			validator.add("tag_statement_id_invalid", "Statement ID must not be empty", nil, nil, nil, nil)
		}
		if !validReviewState(statement.ReviewState) {
			validator.add("tag_statement_review_state_invalid", "Statement review state is not recognised", nil, nil, nil, []string{statement.ID})
		}
		ids := []int64{statement.SubjectTagID, statement.PredicateTagID, statement.ObjectTagID}
		if statement.ContextTagID > 0 {
			ids = append(ids, statement.ContextTagID)
		}
		for _, id := range ids {
			tag, exists := validator.tags[id]
			if !exists {
				validator.add("tag_statement_target_not_found", "Statement refers to a missing Tag", []int64{id}, nil, nil, []string{statement.ID})
			} else if (statement.ReviewState == ReviewProposed || statement.ReviewState == ReviewApproved) && tag.Lifecycle != LifecycleActive {
				validator.add("tag_statement_target_lifecycle_conflict", "Proposed and approved statements require active Tag endpoints", []int64{id}, nil, nil, []string{statement.ID})
			}
		}
		if statement.SubjectTagID == statement.ObjectTagID {
			validator.add("tag_statement_self_relation", "Statement subject and object must differ", []int64{statement.SubjectTagID}, nil, nil, []string{statement.ID})
		}
		if statement.Rank < -1000 || statement.Rank > 1000 {
			validator.add("tag_statement_rank_invalid", "Statement rank must be between -1000 and 1000", nil, nil, nil, []string{statement.ID})
		}
		if statement.ReviewState == ReviewProposed || statement.ReviewState == ReviewApproved {
			identity := fmt.Sprintf("%d/%d/%d/%d/%s", statement.SubjectTagID, statement.PredicateTagID, statement.ObjectTagID, statement.ContextTagID, statement.ReviewState)
			if prior, duplicate := identities[identity]; duplicate {
				validator.add("tag_statement_identity_conflict", "Only one proposed and one approved statement may occupy a semantic identity", nil, nil, nil, []string{prior, statement.ID})
			} else {
				identities[identity] = statement.ID
			}
		}
		if requiresID > 0 && statement.PredicateTagID == requiresID && statement.ReviewState == ReviewApproved {
			requiresEdges[statement.SubjectTagID] = append(requiresEdges[statement.SubjectTagID], statement.ObjectTagID)
			key := fmt.Sprintf("%d/%d", statement.SubjectTagID, statement.ObjectTagID)
			requiresStatement[key] = append(requiresStatement[key], statement.ID)
		}
	}
	for subject := range requiresEdges {
		sort.Slice(requiresEdges[subject], func(i, j int) bool { return requiresEdges[subject][i] < requiresEdges[subject][j] })
	}
	for _, cycle := range directedCycles(requiresEdges) {
		statementIDs := []string{}
		for index := 0; index+1 < len(cycle); index++ {
			key := fmt.Sprintf("%d/%d", cycle[index], cycle[index+1])
			statementIDs = append(statementIDs, requiresStatement[key]...)
		}
		sort.Strings(statementIDs)
		validator.add("tag_requires_cycle", "Approved requires statements must form a directed acyclic graph", append([]int64(nil), cycle...), nil, cycle, statementIDs)
	}
}

func (validator *snapshotValidator) add(code, message string, tagIDs, parentIDs, path []int64, statementIDs []string) {
	validator.violations = append(validator.violations, Violation{
		Code: code, Message: message,
		TagIDs: append([]int64(nil), tagIDs...), ParentTagIDs: append([]int64(nil), parentIDs...),
		PathTagIDs: append([]int64(nil), path...), StatementIDs: append([]string(nil), statementIDs...),
	})
}

func validLifecycle(value Lifecycle) bool {
	switch value {
	case LifecycleActive, LifecycleDeprecated, LifecycleArchived, LifecycleMerged:
		return true
	default:
		return false
	}
}

func validReviewState(value ReviewState) bool {
	switch value {
	case ReviewProposed, ReviewApproved, ReviewRejected, ReviewSuperseded:
		return true
	default:
		return false
	}
}

func overlappingContext(left, right []int64) []int64 {
	left = sortedUniquePositive(left)
	right = sortedUniquePositive(right)
	if len(left) == 0 && len(right) == 0 {
		return []int64{}
	}
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	result := []int64{}
	for i, j := 0, 0; i < len(left) && j < len(right); {
		switch {
		case left[i] == right[j]:
			result = append(result, left[i])
			i++
			j++
		case left[i] < right[j]:
			i++
		default:
			j++
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func sortedUniquePositive(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func sortedTagIDs(tags map[int64]Tag) []int64 {
	ids := make([]int64, 0, len(tags))
	for id := range tags {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func directedCycles(edges map[int64][]int64) [][]int64 {
	state := map[int64]uint8{}
	stack := []int64{}
	positions := map[int64]int{}
	cycles := [][]int64{}
	seen := map[string]struct{}{}
	var visit func(int64)
	visit = func(id int64) {
		state[id] = 1
		positions[id] = len(stack)
		stack = append(stack, id)
		for _, target := range edges[id] {
			switch state[target] {
			case 0:
				visit(target)
			case 1:
				cycle := append([]int64(nil), stack[positions[target]:]...)
				cycle = append(cycle, target)
				key := int64List(cycle)
				if _, duplicate := seen[key]; !duplicate {
					seen[key] = struct{}{}
					cycles = append(cycles, cycle)
				}
			}
		}
		stack = stack[:len(stack)-1]
		delete(positions, id)
		state[id] = 2
	}
	ids := make([]int64, 0, len(edges))
	for id := range edges {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if state[id] == 0 {
			visit(id)
		}
	}
	return cycles
}

func int64List(values []int64) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = fmt.Sprintf("%d", value)
	}
	return strings.Join(parts, ",")
}

func violationKey(value Violation) string {
	return int64List(value.TagIDs) + "|" + int64List(value.ParentTagIDs) + "|" + int64List(value.PathTagIDs) + "|" + strings.Join(value.StatementIDs, ",")
}
