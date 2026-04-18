package install

import "fmt"

// LookupResult holds the outcome of looking up an ID in the installed index.
type LookupResult struct {
	Name  string // display name of the matched package
	Count int    // number of entries that will be affected
}

// countMatchingEntries counts entries matching the predicate across skills and
// agents, returning the total count and the name of the first match.
func countMatchingEntries(skills, agents []InstalledEntry, matchFn func(InstalledEntry) bool) (count int, firstName string) {
	for _, entries := range [][]InstalledEntry{skills, agents} {
		for _, e := range entries {
			if matchFn(e) {
				count++
				if firstName == "" {
					firstName = e.Name
				}
			}
		}
	}
	return count, firstName
}

// LookupByID searches the installed index for a package matching the given ID.
// If the ID matches a bundle, all child entries are included in the count.
// Returns an error if no matching entries are found.
func LookupByID(id string) (*LookupResult, error) {
	idx, err := LoadInstalled()
	if err != nil {
		return nil, err
	}

	bundleFound, bundleName, _ := partitionBundles(idx.Installed.Bundles, id)

	matchFn := func(e InstalledEntry) bool {
		return e.ID == id || (bundleFound && e.BundleID == id)
	}

	count, firstName := countMatchingEntries(idx.Installed.Skills, idx.Installed.Agents, matchFn)

	targetName := bundleName
	if targetName == "" {
		targetName = firstName
	}

	if bundleFound && count == 0 {
		count = 1
	}

	if count == 0 {
		return nil, fmt.Errorf("package with ID %q not found.\nNote: Packages are uninstalled by ID. Try running 'taito list' to find the ID", id)
	}

	return &LookupResult{Name: targetName, Count: count}, nil
}

// GroupResultsByTool groups uninstall results by tool name, preserving order.
func GroupResultsByTool(results []UninstallResult) (specType, name string, toolResults []ToolResult) {
	if len(results) == 0 {
		return "", "", nil
	}

	specType = results[0].SpecType
	name = results[0].Name

	toolMap := make(map[string]*ToolResult)
	var toolOrder []string

	for _, r := range results {
		if r.Tool == "" {
			continue
		}
		tr, ok := toolMap[r.Tool]
		if !ok {
			tr = &ToolResult{Tool: r.Tool}
			toolMap[r.Tool] = tr
			toolOrder = append(toolOrder, r.Tool)
		}
		tr.Paths = append(tr.Paths, r.Path)
	}

	for _, tName := range toolOrder {
		toolResults = append(toolResults, *toolMap[tName])
	}
	return specType, name, toolResults
}

// ToolResult groups paths by tool for uninstall reporting.
type ToolResult struct {
	Tool  string
	Paths []string
}
