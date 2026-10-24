// Package web 总览静态面 embed（T-0067：dist=npm run build 产物——
// gitignore 常态，CI 序 npm build 在 go build 前钦定）。
package web

import "embed"

//go:embed dist
var Dist embed.FS
