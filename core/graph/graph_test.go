package graph

import (
	"reflect"
	"testing"
)

func TestNormalizeNameMatchesRinspaceV1(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"  Differential   Geometry ": "differential geometry",
		"ＫＡＴＥＸ":                      "katex",
		"Straße":                     "strasse",
		"黎曼\n几何":                     "黎曼 几何",
	}
	for input, expected := range cases {
		if actual := NormalizeName(input); actual != expected {
			t.Fatalf("NormalizeName(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestValidateAllowsSameNameInDisjointParentContexts(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Tags = append(snapshot.Tags,
		Tag{ID: 4, DisplayName: "Analysis", NormalizedName: "analysis", ParentTagIDs: []int64{1}, Lifecycle: LifecycleActive, Version: 1},
		Tag{ID: 5, DisplayName: "analysis", NormalizedName: "analysis", ParentTagIDs: []int64{2}, Lifecycle: LifecycleActive, Version: 1},
	)
	if report := Validate(snapshot); !report.Valid {
		t.Fatalf("disjoint name contexts rejected: %#v", report.Violations)
	}
}

func TestValidateAllowsRootAndContextualSameName(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Tags = append(snapshot.Tags,
		Tag{ID: 4, DisplayName: "Analysis", NormalizedName: "analysis", Lifecycle: LifecycleActive, Version: 1},
		Tag{ID: 5, DisplayName: "analysis", NormalizedName: "analysis", ParentTagIDs: []int64{2}, Lifecycle: LifecycleActive, Version: 1},
	)
	if report := Validate(snapshot); !report.Valid {
		t.Fatalf("root and contextual names rejected: %#v", report.Violations)
	}
}

func TestValidateRejectsDuplicateRootName(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Tags = append(snapshot.Tags,
		Tag{ID: 4, DisplayName: "Analysis", NormalizedName: "analysis", Lifecycle: LifecycleActive, Version: 1},
		Tag{ID: 5, DisplayName: "analysis", NormalizedName: "analysis", Lifecycle: LifecycleActive, Version: 1},
	)
	assertViolation(t, Validate(snapshot), "tag_name_context_conflict", []int64{4, 5}, nil)
}

func TestValidateRejectsOverlappingNameContexts(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Tags = append(snapshot.Tags,
		Tag{ID: 4, DisplayName: "Analysis", NormalizedName: "analysis", ParentTagIDs: []int64{1, 2}, Lifecycle: LifecycleActive, Version: 1},
		Tag{ID: 5, DisplayName: "analysis", NormalizedName: "analysis", ParentTagIDs: []int64{2, 3}, Lifecycle: LifecycleActive, Version: 1},
	)
	assertViolation(t, Validate(snapshot), "tag_name_context_conflict", []int64{4, 5}, []int64{2})
}

func TestValidateRejectsParentAndRequiresCycles(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Tags[0].ParentTagIDs = []int64{2}
	snapshot.Tags[1].ParentTagIDs = []int64{1}
	snapshot.Vocabulary = map[string]int64{"requires": 3}
	snapshot.Statements = []Statement{
		{ID: "s1", SubjectTagID: 1, PredicateTagID: 3, ObjectTagID: 2, ReviewState: ReviewApproved},
		{ID: "s2", SubjectTagID: 2, PredicateTagID: 3, ObjectTagID: 1, ReviewState: ReviewApproved},
	}
	report := Validate(snapshot)
	assertViolation(t, report, "tag_parent_cycle", nil, nil)
	assertViolation(t, report, "tag_requires_cycle", nil, nil)
}

func TestValidateAliasAndLifecycleSuccessor(t *testing.T) {
	t.Parallel()
	target := int64(2)
	snapshot := validSnapshot()
	snapshot.Tags[0].Lifecycle = LifecycleMerged
	snapshot.Tags[0].SuccessorTagID = &target
	snapshot.Aliases = []Alias{{DisplayName: "Old name", NormalizedName: "old name", TagID: 2, Active: true}}
	if report := Validate(snapshot); !report.Valid {
		t.Fatalf("valid successor rejected: %#v", report.Violations)
	}
	resolved, path, err := ResolveSuccessor(snapshot, 1)
	if err != nil || resolved.ID != 2 || !reflect.DeepEqual(path, []int64{1, 2}) {
		t.Fatalf("unexpected successor resolution: tag=%#v path=%v err=%v", resolved, path, err)
	}
	alias, err := ResolveAlias(snapshot, "  OLD NAME ")
	if err != nil || alias.ID != 2 {
		t.Fatalf("unexpected alias resolution: tag=%#v err=%v", alias, err)
	}
}

func TestAncestorsPreserveMultiplePaths(t *testing.T) {
	t.Parallel()
	snapshot := validSnapshot()
	snapshot.Tags = append(snapshot.Tags,
		Tag{ID: 4, DisplayName: "Left", NormalizedName: "left", ParentTagIDs: []int64{1}, Lifecycle: LifecycleActive, Version: 1},
		Tag{ID: 5, DisplayName: "Right", NormalizedName: "right", ParentTagIDs: []int64{1}, Lifecycle: LifecycleActive, Version: 1},
		Tag{ID: 6, DisplayName: "Diamond", NormalizedName: "diamond", ParentTagIDs: []int64{4, 5}, Lifecycle: LifecycleActive, Version: 1},
	)
	connections, err := Ancestors(snapshot, 6)
	if err != nil {
		t.Fatal(err)
	}
	want := []Connection{
		{TagID: 6, MinimumDepth: 0, PathCount: 1},
		{TagID: 4, MinimumDepth: 1, PathCount: 1},
		{TagID: 5, MinimumDepth: 1, PathCount: 1},
		{TagID: 1, MinimumDepth: 2, PathCount: 2},
	}
	if !reflect.DeepEqual(connections, want) {
		t.Fatalf("connections = %#v, want %#v", connections, want)
	}
}

func TestValidReportUsesEmptyViolationArray(t *testing.T) {
	t.Parallel()
	report := Validate(validSnapshot())
	if report.Violations == nil || len(report.Violations) != 0 {
		t.Fatalf("valid report violations = %#v, want non-nil empty array", report.Violations)
	}
}

func validSnapshot() Snapshot {
	return Snapshot{
		SchemaVersion: SchemaV1,
		Tags: []Tag{
			{ID: 1, DisplayName: "Mathematics", NormalizedName: "mathematics", Lifecycle: LifecycleActive, Version: 1},
			{ID: 2, DisplayName: "Physics", NormalizedName: "physics", Lifecycle: LifecycleActive, Version: 1},
			{ID: 3, DisplayName: "requires", NormalizedName: "requires", Lifecycle: LifecycleActive, Version: 1},
		},
	}
}

func assertViolation(t *testing.T, report Report, code string, tagIDs, parentIDs []int64) {
	t.Helper()
	for _, violation := range report.Violations {
		if violation.Code != code {
			continue
		}
		if tagIDs != nil && !reflect.DeepEqual(violation.TagIDs, tagIDs) {
			continue
		}
		if parentIDs != nil && !reflect.DeepEqual(violation.ParentTagIDs, parentIDs) {
			continue
		}
		return
	}
	t.Fatalf("missing %s violation in %#v", code, report.Violations)
}
