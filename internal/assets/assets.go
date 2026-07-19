// Package assets bundles static resources shared by the UIs.
package assets

import (
	_ "embed"
)

// IconSVG is the app logo, bundled into the binary. The same file is the snap
// icon (referenced by snap/snapcraft.yaml).
//
//go:embed icon.svg
var IconSVG []byte
