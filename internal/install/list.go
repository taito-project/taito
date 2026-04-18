package install

import (
	"sort"
	"strings"
)

// ListRow represents a single row in the list output table.
type ListRow struct {
	ID       string
	Name     string
	SpecType string
	Tools    string
	Version  string
}

// categorized holds the result of splitting entries into standalone and
// bundle-grouped collections.
type categorized struct {
	standalone      []InstalledEntry
	bundlesChildren map[string][]InstalledEntry
	bundlesByID     map[string]BundleEntry
	bundleOrder     []string
}

// categorizeEntries splits installed entries into standalone items and
// bundle-grouped items, creating synthetic bundle entries for orphaned children.
func categorizeEntries(installed InstalledPackages) categorized {
	c := categorized{
		bundlesChildren: make(map[string][]InstalledEntry),
		bundlesByID:     make(map[string]BundleEntry),
	}

	for _, b := range installed.Bundles {
		c.bundlesByID[b.ID] = b
		c.bundleOrder = append(c.bundleOrder, b.ID)
	}

	classify := func(e InstalledEntry) {
		if e.BundleID == "" {
			c.standalone = append(c.standalone, e)
			return
		}
		bid := e.BundleID
		if _, exists := c.bundlesByID[bid]; !exists {
			c.bundlesByID[bid] = BundleEntry{
				ID:       bid,
				Name:     "unknown-bundle-" + truncateID(bid, 8),
				SpecType: "bundle",
			}
			c.bundleOrder = append(c.bundleOrder, bid)
		}
		c.bundlesChildren[bid] = append(c.bundlesChildren[bid], e)
	}

	for _, e := range installed.Skills {
		classify(e)
	}
	for _, e := range installed.Agents {
		classify(e)
	}
	return c
}

// truncateID shortens an ID to maxLen characters for display.
func truncateID(id string, maxLen int) string {
	if len(id) > maxLen {
		return id[:maxLen]
	}
	return id
}

// buildBundleRows generates tree-formatted rows for all bundles and their children.
func buildBundleRows(bundleOrder []string, bundlesByID map[string]BundleEntry, bundlesChildren map[string][]InstalledEntry, toolDisplayNames map[string]string) [][]string {
	var rows [][]string
	for _, bid := range bundleOrder {
		b := bundlesByID[bid]
		children := bundlesChildren[bid]

		rows = append(rows, []string{
			truncateID(b.ID, 8),
			FormatReference(b.Reference),
			"bundle",
			"",
			b.Version,
		})

		for i, e := range children {
			prefix := " ├─ "
			if i == len(children)-1 {
				prefix = " ╰─ "
			}
			rows = append(rows, EntryRow(e, prefix, toolDisplayNames, true))
		}
	}
	return rows
}

// BuildTreeRows groups installed packages into a tree structure suitable for
// display. Standalone entries come first, followed by bundles with their
// children indented using tree drawing characters.
func BuildTreeRows(installed InstalledPackages, toolDisplayNames map[string]string) [][]string {
	c := categorizeEntries(installed)

	SortEntries(c.standalone)

	sort.Slice(c.bundleOrder, func(i, j int) bool {
		return c.bundlesByID[c.bundleOrder[i]].Name < c.bundlesByID[c.bundleOrder[j]].Name
	})
	for _, bid := range c.bundleOrder {
		children := c.bundlesChildren[bid]
		SortEntries(children)
		c.bundlesChildren[bid] = children
	}

	var rows [][]string
	for _, e := range c.standalone {
		rows = append(rows, EntryRow(e, "", toolDisplayNames, false))
	}

	rows = append(rows, buildBundleRows(c.bundleOrder, c.bundlesByID, c.bundlesChildren, toolDisplayNames)...)
	return rows
}

// FormatReference strips the https:// prefix from a reference string.
func FormatReference(ref string) string {
	return strings.TrimPrefix(ref, "https://")
}

// EntryRow builds a single table row for an installed entry.
func EntryRow(e InstalledEntry, prefix string, toolDisplayNames map[string]string, isChild bool) []string {
	var toolNames []string
	for _, loc := range e.InstallIn {
		name := loc.Tool
		if display, ok := toolDisplayNames[loc.Tool]; ok {
			name = display
		}
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)
	toolsDisplay := strings.Join(toolNames, ", ")

	firstColumn := prefix + FormatReference(e.Reference)
	if isChild {
		firstColumn = prefix + e.Name
	}

	return []string{
		truncateID(e.ID, 8),
		firstColumn,
		e.SpecType,
		toolsDisplay,
		e.Version,
	}
}

// SortEntries sorts installed entries by Name, then SpecType, then tool count.
func SortEntries(entries []InstalledEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		if entries[i].SpecType != entries[j].SpecType {
			return entries[i].SpecType < entries[j].SpecType
		}
		return len(entries[i].InstallIn) < len(entries[j].InstallIn)
	})
}
