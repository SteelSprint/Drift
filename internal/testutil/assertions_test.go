package testutil_test

import (
	"testing"

	"drift/core"
	"drift/internal/testutil"
	"drift/statestore"
)

// TestAssertStateEquals_OrderIndependent: R5 says AssertStateEquals compares
// "after normalizing slice order". Same elements in different order should
// pass, not fail.
func TestAssertStateEquals_OrderIndependent(t *testing.T) {
	specA := testutil.NewSpec("m.a", "hash-a")
	specB := testutil.NewSpec("m.b", "hash-b")
	markerX := testutil.NewMarker("mx", "hash-x")
	markerY := testutil.NewMarker("my", "hash-y")
	edge1 := testutil.NewLink("m.a", "mx")
	edge2 := testutil.NewLink("m.b", "my")

	// got and want have the same elements but in different order.
	got := statestore.State{
		Specs:   []core.Spec{specA, specB},
		Markers: []core.Marker{markerX, markerY},
		Edges:   []core.Edge{edge1, edge2},
	}
	want := statestore.State{
		Specs:   []core.Spec{specB, specA}, // reversed
		Markers: []core.Marker{markerY, markerX}, // reversed
		Edges:   []core.Edge{edge2, edge1}, // reversed
	}

	// Should NOT fatal — same elements, different order.
	AssertStateEqualsNoFatal(t, got, want)
}

// AssertStateEqualsNoFatal wraps AssertStateEquals but converts a Fatal into
// a real Error so the test reports the failure instead of stopping.
func AssertStateEqualsNoFatal(t *testing.T, got, want statestore.State) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AssertStateEquals fatal on order-differing states (spec R5 says normalize order): %v", r)
		}
	}()
	testutil.AssertStateEquals(t, got, want)
}
