package mergerfs

import "strings"

// BuildMountOptions renders mergerfs mount options with safe defaults.
func BuildMountOptions(categoryCreate, minFreeSpace, extra string) string {
	parts := []string{"defaults", "allow_other"}
	if categoryCreate != "" {
		parts = append(parts, "category.create="+categoryCreate)
	}
	if minFreeSpace != "" {
		parts = append(parts, "minfreespace="+minFreeSpace)
	}
	if extra != "" {
		parts = append(parts, extra)
	}
	return strings.Join(parts, ",")
}
