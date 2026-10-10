package graph

import (
	"errors"
	"sort"
)

var (
	ErrTagNotFound   = errors.New("tag not found")
	ErrAliasNotFound = errors.New("alias not found")
	ErrInvalidGraph  = errors.New("tag graph is invalid")
)

// Ancestors returns the transitive parent projection for one Tag. The result
// includes the Tag itself at depth zero, matching the persisted closure model.
func Ancestors(snapshot Snapshot, tagID int64) ([]Connection, error) {
	index, err := validatedIndex(snapshot)
	if err != nil {
		return nil, err
	}
	if _, exists := index.tags[tagID]; !exists {
		return nil, ErrTagNotFound
	}
	return connections(tagID, index.parents), nil
}

// Descendants returns the inverse transitive classification projection.
func Descendants(snapshot Snapshot, tagID int64) ([]Connection, error) {
	index, err := validatedIndex(snapshot)
	if err != nil {
		return nil, err
	}
	if _, exists := index.tags[tagID]; !exists {
		return nil, ErrTagNotFound
	}
	children := make(map[int64][]int64, len(index.tags))
	for childID, parentIDs := range index.parents {
		for _, parentID := range parentIDs {
			children[parentID] = append(children[parentID], childID)
		}
	}
	for id := range children {
		sort.Slice(children[id], func(i, j int) bool { return children[id][i] < children[id][j] })
	}
	return connections(tagID, children), nil
}

// ResolveAlias resolves an active alias to its stable Tag identity.
func ResolveAlias(snapshot Snapshot, name string) (Tag, error) {
	index, err := validatedIndex(snapshot)
	if err != nil {
		return Tag{}, err
	}
	normalized := NormalizeName(name)
	for _, alias := range snapshot.Aliases {
		if alias.Active && alias.NormalizedName == normalized {
			return index.tags[alias.TagID], nil
		}
	}
	return Tag{}, ErrAliasNotFound
}

// ResolveSuccessor follows lifecycle redirects and returns the canonical Tag
// together with the complete stable-ID path.
func ResolveSuccessor(snapshot Snapshot, tagID int64) (Tag, []int64, error) {
	index, err := validatedIndex(snapshot)
	if err != nil {
		return Tag{}, nil, err
	}
	current, exists := index.tags[tagID]
	if !exists {
		return Tag{}, nil, ErrTagNotFound
	}
	path := []int64{tagID}
	for current.SuccessorTagID != nil && *current.SuccessorTagID > 0 {
		current = index.tags[*current.SuccessorTagID]
		path = append(path, current.ID)
	}
	return current, path, nil
}

type graphIndex struct {
	tags    map[int64]Tag
	parents map[int64][]int64
}

func validatedIndex(snapshot Snapshot) (graphIndex, error) {
	report := Validate(snapshot)
	if !report.Valid {
		return graphIndex{}, ErrInvalidGraph
	}
	index := graphIndex{tags: make(map[int64]Tag, len(snapshot.Tags)), parents: make(map[int64][]int64, len(snapshot.Tags))}
	for _, tag := range snapshot.Tags {
		index.tags[tag.ID] = tag
		index.parents[tag.ID] = append([]int64(nil), tag.ParentTagIDs...)
	}
	return index, nil
}

func connections(start int64, edges map[int64][]int64) []Connection {
	reachable := map[int64]struct{}{start: {}}
	discovery := []int64{start}
	for len(discovery) > 0 {
		current := discovery[0]
		discovery = discovery[1:]
		for _, next := range edges[current] {
			if _, seen := reachable[next]; seen {
				continue
			}
			reachable[next] = struct{}{}
			discovery = append(discovery, next)
		}
	}

	indegree := make(map[int64]int, len(reachable))
	for current := range reachable {
		for _, next := range edges[current] {
			if _, ok := reachable[next]; ok {
				indegree[next]++
			}
		}
	}
	queue := []int64{}
	for id := range reachable {
		if indegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i] < queue[j] })
	depth := map[int64]int{start: 0}
	paths := map[int64]int64{start: 1}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range edges[current] {
			if _, ok := reachable[next]; !ok {
				continue
			}
			candidateDepth := depth[current] + 1
			priorDepth, seen := depth[next]
			paths[next] += paths[current]
			if !seen || candidateDepth < priorDepth {
				depth[next] = candidateDepth
			}
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
				sort.Slice(queue, func(i, j int) bool { return queue[i] < queue[j] })
			}
		}
	}
	result := make([]Connection, 0, len(depth))
	for id, minimumDepth := range depth {
		result = append(result, Connection{TagID: id, MinimumDepth: minimumDepth, PathCount: paths[id]})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].MinimumDepth != result[j].MinimumDepth {
			return result[i].MinimumDepth < result[j].MinimumDepth
		}
		return result[i].TagID < result[j].TagID
	})
	return result
}
