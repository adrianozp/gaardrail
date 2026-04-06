package web

import (
	"embed"
	_ "embed"
)

//go:embed index.html
var IndexHTML []byte

//go:embed favicon
var Favicons embed.FS
