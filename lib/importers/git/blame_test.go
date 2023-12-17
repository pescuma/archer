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

	t.Equal([]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}},
		lo.Map(result, func(d *mergedDiff, _ int) linediff.Diff { return d.Diff }))
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

	t.Equal([]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}},
		lo.Map(result, func(d *mergedDiff, _ int) linediff.Diff { return d.Diff }))
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

	t.Equal([]linediff.Diff{{Type: linediff.DiffEqual, Lines: 10}},
		lo.Map(result, func(d *mergedDiff, _ int) linediff.Diff { return d.Diff }))
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
	},
		lo.Map(result, func(d *mergedDiff, _ int) linediff.Diff { return d.Diff }))
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
	changed, notChanged := computeAffected(newRangesWithLines(10), []*mergedDiff{
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

	t.Equal(newRanges(newRange(1, 2, 0)), changed)

	t.Equal(newRanges(newRange(0, 0, 0), newRange(3, 9, 0)),
		newRanges(lo.Map(notChanged, func(r *linesRangeWithSource, _ int) linesRange { return r.linesRange })...))
}

func TestUpdateRanges(t *testing.T) {
	testgroup.RunInParallel(t, &UpdateRangesTests{})
}

type UpdateRangesTests struct {
}

func (g *UpdateRangesTests) OneInsert(t *testgroup.T) {
	ranges := newRanges(newRange(2, 9, 0))
	diffs := []linediff.Diff{
		{Type: linediff.DiffInsert, Lines: 1},
		{Type: linediff.DiffEqual, Lines: 9},
	}

	r := updateRanges(ranges, diffs)

	t.Equal(newRanges(newRange(1, 8, 1)), r)
}

func (g *UpdateRangesTests) OneDelete(t *testgroup.T) {
	ranges := newRanges(newRange(2, 9, 0))
	diffs := []linediff.Diff{
		{Type: linediff.DiffDelete, Lines: 1},
		{Type: linediff.DiffEqual, Lines: 10},
	}

	r := updateRanges(ranges, diffs)

	t.Equal(newRanges(newRange(3, 10, -1)), r)
}

func (g *UpdateRangesTests) OneInsertOneDelete(t *testgroup.T) {
	ranges := newRanges(
		newRange(0, 0, 0),
		newRange(5, 6, 0),
		newRange(13, 15, 0),
	)
	diffs := []linediff.Diff{
		{Type: linediff.DiffEqual, Lines: 5},
		{Type: linediff.DiffDelete, Lines: 1},
		{Type: linediff.DiffEqual, Lines: 5},
		{Type: linediff.DiffInsert, Lines: 2},
		{Type: linediff.DiffEqual, Lines: 5},
	}

	r := updateRanges(ranges, diffs)

	t.Equal(newRanges(
		newRange(0, 0, 0),
		newRange(6, 7, -1),
		newRange(12, 14, 1),
	), r)
}

func TestMergeRanges(t *testing.T) {
	testgroup.RunInParallel(t, &MergeRangesTests{})
}

type MergeRangesTests struct {
}

func (g *MergeRangesTests) Same(t *testgroup.T) {
	r1 := newRangesWithLines(10)
	r2 := newRangesWithLines(10)

	r := r1.merge(r2)

	t.Equal(newRanges(newRange(0, 9, 0)), r)
}

func (g *MergeRangesTests) IntersectionSameOffsets(t *testgroup.T) {
	r1 := newRanges(newRange(0, 9, 0))
	r2 := newRanges(newRange(8, 20, 0))

	r := r1.merge(r2)

	t.Equal(newRanges(newRange(0, 20, 0)), r)
}

func (g *MergeRangesTests) IntersectionDifferentOffsets(t *testgroup.T) {
	r1 := newRanges(newRange(0, 9, 0))
	r2 := newRanges(newRange(8, 20, 1))

	r := r1.merge(r2)

	t.Equal(newRanges(
		newRange(0, 7, 0),
		newRange(8, 9, 0, 1),
		newRange(10, 20, 1),
	),
		r)
}
