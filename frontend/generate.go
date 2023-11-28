package frontend

import "embed"

//go:generate npm run build

//go:embed dist/assets
var Assets embed.FS

//go:embed dist/index.html
var Index embed.FS
