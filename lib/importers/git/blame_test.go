package git

import (
	"testing"

	"github.com/bloomberg/go-testgroup"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hashicorp/go-set/v2"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/linediff"
)

func TestMergeParentChanges(t *testing.T) {
	testgroup.RunInParallel(t, &MergeParentChangesTests{})
}

type MergeParentChangesTests struct {
}

func (g *MergeParentChangesTests) NoChanges(t *testgroup.T) {
	parents := g.createParents(
		[]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}},
		[]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}},
	)

	result := mergeParentChanges(parents)

	t.Equal([]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}}, result)
}

func (g *MergeParentChangesTests) IgnoreDeletes(t *testgroup.T) {
	parents := g.createParents(
		[]linediff.Diff{
			{Type: linediff.DiffEqual, Lines: 2},
			{Type: linediff.DiffDelete, Lines: 1},
			{Type: linediff.DiffEqual, Lines: 8},
		},
		[]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}},
	)

	result := mergeParentChanges(parents)

	t.Equal([]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}}, result)
}

func (g *MergeParentChangesTests) IgnoreInsertWhenOtherSideIsEqual(t *testgroup.T) {
	parents := g.createParents(
		[]linediff.Diff{
			{Type: linediff.DiffEqual, Lines: 2},
			{Type: linediff.DiffInsert, Lines: 1},
			{Type: linediff.DiffEqual, Lines: 7},
		},
		[]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}},
	)

	result := mergeParentChanges(parents)

	t.Equal([]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}}, result)
}

func (g *MergeParentChangesTests) KeepInsertWhenSameFromBothSides(t *testgroup.T) {
	parents := g.createParents(
		[]linediff.Diff{
			{Type: linediff.DiffInsert, Lines: 10},
		},
		[]linediff.Diff{
			{Type: linediff.DiffEqual, Lines: 2},
			{Type: linediff.DiffInsert, Lines: 1},
			{Type: linediff.DiffEqual, Lines: 7},
		},
	)

	result := mergeParentChanges(parents)

	t.Equal([]linediff.Diff{
		{Type: linediff.DiffEqual, Lines: 2},
		{Type: linediff.DiffInsert, Lines: 1},
		{Type: linediff.DiffEqual, Lines: 7},
	}, result)
}

func (g *MergeParentChangesTests) createParents(diffs ...[]linediff.Diff) []*parentItem {
	var parents []*parentItem
	for _, d := range diffs {
		parents = append(parents, &parentItem{
			Diff: d,
		})
	}
	return parents
}

func TestComputeAffected(t *testing.T) {
	testgroup.RunInParallel(t, &ComputeAffectedTests{})
}

type ComputeAffectedTests struct {
}

func (g *ComputeAffectedTests) OneChange(t *testgroup.T) {
	changed, notChanged := computeAffected(newLinesRangesAll(10), []*mergedDiff{
		{
			Diff:    linediff.Diff{Type: linediff.DiffEqual, Lines: 1},
			sources: set.New[plumbing.Hash](1),
		},
		{
			Diff:    linediff.Diff{Type: linediff.DiffInsert, Lines: 2},
			sources: set.New[plumbing.Hash](1),
		},
		{
			Diff:    linediff.Diff{Type: linediff.DiffEqual, Lines: 7},
			sources: set.New[plumbing.Hash](1),
		},
	})

	t.Equal([]linesRange{newLinesRange(1, 2, 0)}, changed)

	t.Equal([]linesRange{newLinesRange(0, 0, 0), newLinesRange(3, 9, 0)},
		lo.Map(notChanged, func(r *linesRangeWithSource, _ int) linesRange { return r.linesRange }))
}

func TestUpdateRanges(t *testing.T) {
	testgroup.RunInParallel(t, &UpdateRangesTests{})
}

type UpdateRangesTests struct {
}

func (g *UpdateRangesTests) OneInsert(t *testgroup.T) {
	ranges := []linesRange{newLinesRange(2, 9, 0)}
	diffs := []linediff.Diff{
		{Type: linediff.DiffInsert, Lines: 1},
		{Type: linediff.DiffEqual, Lines: 9},
	}

	r := updateRanges(ranges, diffs)

	t.Equal([]linesRange{newLinesRange(1, 8, 1)}, r)
}

func (g *UpdateRangesTests) OneDelete(t *testgroup.T) {
	ranges := []linesRange{newLinesRange(2, 9, 0)}
	diffs := []linediff.Diff{
		{Type: linediff.DiffDelete, Lines: 1},
		{Type: linediff.DiffEqual, Lines: 10},
	}

	r := updateRanges(ranges, diffs)

	t.Equal([]linesRange{newLinesRange(3, 10, -1)}, r)
}

func (g *UpdateRangesTests) OneInsertOneDelete(t *testgroup.T) {
	ranges := []linesRange{
		newLinesRange(0, 0, 0),
		newLinesRange(5, 6, 0),
		newLinesRange(13, 15, 0),
	}
	diffs := []linediff.Diff{
		{Type: linediff.DiffEqual, Lines: 5},
		{Type: linediff.DiffDelete, Lines: 1},
		{Type: linediff.DiffEqual, Lines: 5},
		{Type: linediff.DiffInsert, Lines: 2},
		{Type: linediff.DiffEqual, Lines: 5},
	}

	r := updateRanges(ranges, diffs)

	t.Equal([]linesRange{
		newLinesRange(0, 0, 0),
		newLinesRange(6, 7, -1),
		newLinesRange(12, 14, 1),
	}, r)
}
