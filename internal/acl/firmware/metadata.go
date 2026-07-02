package firmware

import (
	"strings"

	properties "github.com/arduino/go-properties-orderedmap"
)

// ResolveTargetChip returns the most specific target-chip metadata available.
//
// build.mcu is the canonical source when the build system provides it. The
// fallback is used when the build properties do not expose a chip value but the
// caller has a reasonable board-level identifier.
func ResolveTargetChip(props *properties.Map, fallback string) string {
	if props != nil {
		if chip := strings.TrimSpace(props.Get("build.mcu")); chip != "" {
			return chip
		}
	}
	return strings.TrimSpace(fallback)
}
