// Package migrations —— 迁移文件序进嵌入（schema_migrations 基线制，架构约束 §1）。
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
