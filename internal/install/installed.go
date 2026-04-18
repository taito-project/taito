package install

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/taito-project/taito/internal/config"
)

// InstallLocation tracks a tool and path where a package is installed.
type InstallLocation struct {
	Tool string `json:"tool"`
	Path string `json:"path"`
}

// InstalledEntry tracks a single skill or agent installed into one or more tool directories.
type InstalledEntry struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	SpecType          string            `json:"specType"`
	Version           string            `json:"version,omitempty"`
	Reference         string            `json:"reference,omitempty"`
	InternalReference string            `json:"internalReference,omitempty"`
	BundleID          string            `json:"bundleId,omitempty"`
	InstallIn         []InstallLocation `json:"installIn"`
	InstalledAt       string            `json:"installedAt"`
}

// BundleEntry tracks a bundle installation.
type BundleEntry struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	SpecType          string `json:"specType"`
	Version           string `json:"version,omitempty"`
	Reference         string `json:"reference,omitempty"`
	InternalReference string `json:"internalReference,omitempty"`
}

// InstalledPackages groups installed entries by type.
type InstalledPackages struct {
	Skills  []InstalledEntry `json:"skills"`
	Agents  []InstalledEntry `json:"agents"`
	Bundles []BundleEntry    `json:"bundles"`
}

const IndexSpecVersion = "1.0.0"

// InstalledIndex is the top-level structure for installed.json.
type InstalledIndex struct {
	Version   string            `json:"version"`
	Installed InstalledPackages `json:"installed"`
}

// installedMu serializes concurrent reads/writes to the installed index.
var installedMu sync.Mutex

// installedPath returns the path to installed.json in the config directory.
func installedPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "installed.json"), nil
}

// LoadInstalled reads the installed.json file. Returns an empty index (not an
// error) if the file does not exist.
func LoadInstalled() (*InstalledIndex, error) {
	installedMu.Lock()
	defer installedMu.Unlock()

	return loadInstalledLocked()
}

// loadInstalledLocked reads the index without acquiring the mutex.
func loadInstalledLocked() (*InstalledIndex, error) {
	p, err := installedPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &InstalledIndex{
				Version: IndexSpecVersion,
				Installed: InstalledPackages{
					Skills:  []InstalledEntry{},
					Agents:  []InstalledEntry{},
					Bundles: []BundleEntry{},
				},
			}, nil
		}
		return nil, fmt.Errorf("read installed index: %w", err)
	}

	var idx InstalledIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		// On unmarshal error, assume corrupt or old format, return fresh index.
		fmt.Fprintf(os.Stderr, "warning: installed index is corrupt, starting fresh: %v\n", err)
		return &InstalledIndex{
			Version: IndexSpecVersion,
			Installed: InstalledPackages{
				Skills:  []InstalledEntry{},
				Agents:  []InstalledEntry{},
				Bundles: []BundleEntry{},
			},
		}, nil
	}

	if idx.Installed.Skills == nil {
		idx.Installed.Skills = []InstalledEntry{}
	}
	if idx.Installed.Agents == nil {
		idx.Installed.Agents = []InstalledEntry{}
	}
	if idx.Installed.Bundles == nil {
		idx.Installed.Bundles = []BundleEntry{}
	}

	return &idx, nil
}

// SaveInstalled writes the installed index to disk.
func SaveInstalled(idx *InstalledIndex) error {
	installedMu.Lock()
	defer installedMu.Unlock()

	return saveInstalledLocked(idx)
}

// saveInstalledLocked writes the index without acquiring the mutex.
func saveInstalledLocked(idx *InstalledIndex) error {
	p, err := installedPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal installed index: %w", err)
	}

	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("write installed index: %w", err)
	}
	return nil
}

// generateHexID creates an 8-character random hex string.
func generateHexID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x", b)
}

// isIDUnique checks if the given ID is already used across skills, agents, and bundles.
func isIDUnique(idx *InstalledIndex, id string) bool {
	for _, s := range idx.Installed.Skills {
		if s.ID == id {
			return false
		}
	}
	for _, a := range idx.Installed.Agents {
		if a.ID == id {
			return false
		}
	}
	for _, b := range idx.Installed.Bundles {
		if b.ID == id {
			return false
		}
	}
	return true
}

// generateUniqueID generates a UUID that is guaranteed to be unique in the index.
func generateUniqueID(idx *InstalledIndex) string {
	for {
		id := generateHexID()
		if isIDUnique(idx, id) {
			return id
		}
	}
}

// mergeInstallLocations merges two InstallLocation slices, deduplicating by Tool.
// Locations from newer take precedence over existing.
func mergeInstallLocations(existing, newer []InstallLocation) []InstallLocation {
	locMap := make(map[string]InstallLocation, len(existing)+len(newer))
	for _, loc := range existing {
		locMap[loc.Tool] = loc
	}
	for _, loc := range newer {
		locMap[loc.Tool] = loc
	}
	merged := make([]InstallLocation, 0, len(locMap))
	for _, loc := range locMap {
		merged = append(merged, loc)
	}
	return merged
}

// ensureID preserves an existing ID, or generates a new unique one.
func ensureID(idx *InstalledIndex, existingID string) string {
	if existingID != "" {
		return existingID
	}
	return generateUniqueID(idx)
}

// ensureUniqueID returns the provided ID if it is non-empty and unique,
// otherwise generates a new unique one.
func ensureUniqueID(idx *InstalledIndex, id string) string {
	if id != "" && isIDUnique(idx, id) {
		return id
	}
	return generateUniqueID(idx)
}

// filterToolFromEntries removes a specific tool from entries matching a name.
// Entries that have no remaining InstallIn locations are dropped entirely.
func filterToolFromEntries(entries []InstalledEntry, name, tool string) []InstalledEntry {
	result := make([]InstalledEntry, 0, len(entries))
	for _, e := range entries {
		if e.Name != name {
			result = append(result, e)
			continue
		}
		filtered := make([]InstallLocation, 0, len(e.InstallIn))
		for _, loc := range e.InstallIn {
			if loc.Tool != tool {
				filtered = append(filtered, loc)
			}
		}
		e.InstallIn = filtered
		if len(e.InstallIn) > 0 {
			result = append(result, e)
		}
	}
	return result
}

// partitionEntries splits entries into matched and kept slices based on a predicate.
func partitionEntries(entries []InstalledEntry, matchFn func(InstalledEntry) bool) (matched, kept []InstalledEntry) {
	for _, e := range entries {
		if matchFn(e) {
			matched = append(matched, e)
		} else {
			kept = append(kept, e)
		}
	}
	return matched, kept
}

// entriesToResults converts matched entries into UninstallResults,
// removing their directories from disk.
func entriesToResults(entries []InstalledEntry) []UninstallResult {
	var results []UninstallResult
	for _, e := range entries {
		for _, loc := range e.InstallIn {
			if loc.Path != "" {
				_ = os.RemoveAll(loc.Path)
			}
			results = append(results, UninstallResult{
				Name:     e.Name,
				SpecType: e.SpecType,
				Tool:     ToolDisplayName(loc.Tool),
				Path:     loc.Path,
			})
		}
	}
	return results
}

// UpsertEntry loads the index, upserts an entry keyed by (name, specType) into either
// Skills or Agents, merges its InstallIn array, and saves it back.
func UpsertEntry(entry InstalledEntry) error {
	installedMu.Lock()
	defer installedMu.Unlock()

	idx, err := loadInstalledLocked()
	if err != nil {
		return err
	}

	entry.InstalledAt = time.Now().UTC().Format(time.RFC3339)

	targetList := &idx.Installed.Skills
	if entry.SpecType == "agent" {
		targetList = &idx.Installed.Agents
	}

	found := false
	for i, e := range *targetList {
		if e.Name == entry.Name && e.SpecType == entry.SpecType && e.InternalReference == entry.InternalReference {
			entry.InstallIn = mergeInstallLocations(e.InstallIn, entry.InstallIn)
			entry.ID = ensureID(idx, e.ID)
			(*targetList)[i] = entry
			found = true
			break
		}
	}
	if !found {
		entry.ID = ensureUniqueID(idx, entry.ID)
		*targetList = append(*targetList, entry)
	}

	return saveInstalledLocked(idx)
}

// UpsertBundle adds or updates a bundle in the Bundles list.
// Returns the bundle's ID.
func UpsertBundle(bundle BundleEntry) (string, error) {
	installedMu.Lock()
	defer installedMu.Unlock()

	idx, err := loadInstalledLocked()
	if err != nil {
		return "", err
	}

	found := false
	for i, b := range idx.Installed.Bundles {
		if b.Name == bundle.Name && b.InternalReference == bundle.InternalReference {
			bundle.ID = ensureID(idx, b.ID)
			idx.Installed.Bundles[i] = bundle
			found = true
			break
		}
	}
	if !found {
		bundle.ID = ensureUniqueID(idx, bundle.ID)
		idx.Installed.Bundles = append(idx.Installed.Bundles, bundle)
	}

	if err := saveInstalledLocked(idx); err != nil {
		return "", err
	}

	return bundle.ID, nil
}

// RemoveEntry removes all entries matching the given (name, tool) pair from
// Skills and Agents. If the entry is installed in multiple tools, it only
// removes the specified tool from InstallIn. If InstallIn becomes empty,
// the entire entry is removed.
func RemoveEntry(name, tool string) error {
	installedMu.Lock()
	defer installedMu.Unlock()

	idx, err := loadInstalledLocked()
	if err != nil {
		return err
	}

	idx.Installed.Skills = filterToolFromEntries(idx.Installed.Skills, name, tool)
	idx.Installed.Agents = filterToolFromEntries(idx.Installed.Agents, name, tool)

	return saveInstalledLocked(idx)
}

// UninstallResult holds the outcome of removing a single entry.
type UninstallResult struct {
	Name     string
	SpecType string
	Tool     string
	Path     string
}

// Uninstall removes all installed entries matching the given ID. If the ID
// matches a bundle, all child entries that were installed by that bundle are
// also removed. For each matching entry, the installed directory is deleted
// from disk and the entry is removed from installed.json.
//
// Returns the list of removed entries or an error. If no entries match, an
// error is returned.
func Uninstall(id string) ([]UninstallResult, error) {
	installedMu.Lock()
	defer installedMu.Unlock()

	idx, err := loadInstalledLocked()
	if err != nil {
		return nil, err
	}

	// Check if this is a bundle.
	bundleRemoved, bundleName, keepBundles := partitionBundles(idx.Installed.Bundles, id)

	// Partition skills and agents — match by direct ID or bundle membership.
	matchFn := func(e InstalledEntry) bool {
		return e.ID == id || (bundleRemoved && e.BundleID == id)
	}
	removedSkills, keepSkills := partitionEntries(idx.Installed.Skills, matchFn)
	removedAgents, keepAgents := partitionEntries(idx.Installed.Agents, matchFn)

	toRemove := append(removedSkills, removedAgents...)
	if len(toRemove) == 0 && !bundleRemoved {
		return nil, fmt.Errorf("package with ID %q not found.\nNote: Packages are uninstalled by ID. Try running 'taito list' to find the ID", id)
	}

	// Remove directories from disk and build results.
	results := entriesToResults(toRemove)

	// Add bundle to results if it had no children but existed.
	if len(toRemove) == 0 && bundleRemoved {
		results = append(results, UninstallResult{
			Name:     bundleName,
			SpecType: "bundle",
		})
	}

	// Update the index.
	idx.Installed.Skills = keepSkills
	idx.Installed.Agents = keepAgents
	idx.Installed.Bundles = keepBundles

	if err := saveInstalledLocked(idx); err != nil {
		return results, fmt.Errorf("update installed index: %w", err)
	}

	return results, nil
}

// partitionBundles splits bundles by ID, returning whether the bundle was found,
// its name, and the remaining bundles.
func partitionBundles(bundles []BundleEntry, id string) (found bool, name string, kept []BundleEntry) {
	for _, b := range bundles {
		if b.ID == id {
			found = true
			name = b.Name
		} else {
			kept = append(kept, b)
		}
	}
	return found, name, kept
}
