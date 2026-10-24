// Package web 总览静态面 embed（T-0067：设计稿活页=钦定视觉锚直开径）。
package web

import "embed"

//go:embed page
var Page embed.FS
