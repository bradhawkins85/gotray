package menu

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/example/gotray/internal/config"
)

// EnsureSequentialOrder assigns deterministic order values for menu items.
func EnsureSequentialOrder(items *[]config.MenuItem) {
	if items == nil {
		return
	}
	groups := make(map[string][]int)
	for idx := range *items {
		parent := (*items)[idx].ParentID
		groups[parent] = append(groups[parent], idx)
	}

	for _, indices := range groups {
		sort.SliceStable(indices, func(i, j int) bool {
			return (*items)[indices[i]].Order < (*items)[indices[j]].Order
		})
		for pos, idx := range indices {
			(*items)[idx].Order = (pos + 1) * 10
		}
	}
}

// GenerateID produces a stable identifier using sequential integers.
func GenerateID(items []config.MenuItem, parentID string, itemType config.MenuItemType) string {
	if parentID == "" {
		maxVal := 0
		for _, item := range items {
			if item.ParentID != "" {
				continue
			}
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

	siblings := filterByParent(items, parentID)
	next := 0
	for _, sibling := range siblings {
		suffix := lastSegment(sibling.ID)
		if suffix == "" {
			continue
		}
		if v, err := strconv.Atoi(suffix); err == nil {
			if v > next {
				next = v
			}
		}
	}

	if itemType == config.MenuItemMenu {
		if next <= 0 {
			next = 10
		} else {
			next = ((next / 10) + 1) * 10
		}
	} else {
		next++
		if next <= 0 {
			next = 1
		}
	}

	suffix := strconv.Itoa(next)
	return parentID + "." + suffix
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

// ValidateParent ensures the requested parent relationship is supported.
func ValidateParent(all []config.MenuItem, item config.MenuItem) error {
	if item.ParentID == "" {
		return nil
	}

	if item.ParentID == item.ID {
		return fmt.Errorf("item %s cannot reference itself as parent", item.ID)
	}

	parent := findItemByID(all, item.ParentID)
	if parent == nil {
		return fmt.Errorf("parent id %s not found", item.ParentID)
	}

	if item.Type == config.MenuItemMenu {
		last := lastSegment(item.ParentID)
		if last == "" {
			return fmt.Errorf("parent id %s is not valid for menu items", item.ParentID)
		}
		value, err := strconv.Atoi(last)
		if err != nil || value%10 != 0 {
			return fmt.Errorf("parent id %s must resolve to a multiple of 10", item.ParentID)
		}
	}

	return nil
}

func filterByParent(items []config.MenuItem, parentID string) []config.MenuItem {
	out := make([]config.MenuItem, 0)
	for _, item := range items {
		if item.ParentID == parentID {
			out = append(out, item)
		}
	}
	return out
}

func findItemByID(items []config.MenuItem, id string) *config.MenuItem {
	for idx := range items {
		if items[idx].ID == id {
			return &items[idx]
		}
	}
	return nil
}

func lastSegment(id string) string {
	if id == "" {
		return ""
	}
	parts := strings.Split(id, ".")
	return parts[len(parts)-1]
}
