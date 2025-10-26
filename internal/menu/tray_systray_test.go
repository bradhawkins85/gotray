//go:build cgo || windows
// +build cgo windows

package menu

import (
	"testing"

	"github.com/example/gotray/internal/config"
)

func TestGroupByParentSortsDescendingOrder(t *testing.T) {
	items := []config.MenuItem{
		{ID: "10", Order: 10},
		{ID: "20", Order: 20},
		{ID: "30", Order: 30},
	}

	grouped := groupByParent(items)
	root := grouped[""]

	if len(root) != 3 {
		t.Fatalf("expected 3 items, got %d", len(root))
	}

	expected := []int{30, 20, 10}
	for idx, want := range expected {
		if root[idx].Order != want {
			t.Fatalf("position %d expected order %d got %d", idx, want, root[idx].Order)
		}
	}
}
