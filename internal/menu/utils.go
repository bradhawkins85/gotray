package menu

import (
	"strconv"

	"github.com/example/gotray/internal/config"
)

// EnsureSequentialOrder assigns deterministic order values for menu items.
func EnsureSequentialOrder(items *[]config.MenuItem) {
	if items == nil {
		return
	}
	for i := range *items {
		(*items)[i].Order = (i + 1) * 10
	}
}

// GenerateID produces a stable identifier using sequential integers.
func GenerateID(items []config.MenuItem) string {
	maxVal := 0
	for _, item := range items {
		if v, err := strconv.Atoi(item.ID); err == nil {
			if v > maxVal {
				maxVal = v
			}
		}
	}

	if maxVal <= 0 {
		return "10"
	}

	next := ((maxVal / 10) + 1) * 10
	if next <= 0 {
		next = 10
	}
	return strconv.Itoa(next)
}

// InsertItem injects an item at the requested index and shifts subsequent entries.
func InsertItem(items []config.MenuItem, index int, item config.MenuItem) []config.MenuItem {
	if index < 0 {
		index = 0
	}
	if index > len(items) {
		index = len(items)
	}

	items = append(items, config.MenuItem{})
	copy(items[index+1:], items[index:])
	items[index] = item
	return items
}

// RemoveIndex deletes the element at index when in bounds.
func RemoveIndex(items []config.MenuItem, index int) []config.MenuItem {
	if index < 0 || index >= len(items) {
		return items
	}
	return append(items[:index], items[index+1:]...)
}
