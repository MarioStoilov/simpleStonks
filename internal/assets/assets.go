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

// BellPlusSVG is the add-price-alert bell icon: a white outline bell with a
// plus sign at its side (font-independent, unlike the bell emoji).
//
//go:embed bell-plus.svg
var BellPlusSVG []byte

// OfflineSVG is the connection-lost indicator: an unplugged connector pair.
//
//go:embed offline.svg
var OfflineSVG []byte
