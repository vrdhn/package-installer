package embed

import "embed"

//go:embed cli.def
var CLIDef string

//go:embed recipes/*.star
var Recipes embed.FS
