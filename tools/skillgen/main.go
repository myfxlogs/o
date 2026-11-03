// skillgen —— T-0314 语义类消费腿（canon 案→skills/econ-<slug>/SKILL.md
// 人工裁定后命令行触发；只写 skills/econ-* 越域拒；frontmatter 三行照
// ctx-sources 形——返修① R2）。
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// usage: skillgen <slug> <lesson_id> —— 读 fin_econ_lesson 行→SKILL.md 落件。
func main() {
	if len(os.Args) < 3 || !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(os.Args[1]) {
		fmt.Fprintln(os.Stderr, "usage: skillgen <slug> <lesson_id>")
		os.Exit(2)
	}
	slug, id := os.Args[1], os.Args[2]
	if !strings.HasPrefix(slug, "econ-") {
		fmt.Fprintln(os.Stderr, "slug 须 econ-* 前缀")
		os.Exit(2)
	}
	dsn := os.Getenv("FIN_PG_DSN")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	var kind, ref, nums string
	if err := pool.QueryRow(context.Background(),
		`SELECT kind,entity_ref,numbers::text FROM fin_econ_lesson WHERE id=$1`, id).
		Scan(&kind, &ref, &nums); err != nil {
		panic(err)
	}
	dir := "skills/" + slug
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	doc := fmt.Sprintf(`---
name: %s
description: 经济案 canon（T-0314 裁定生成件）——kind=%s ref=%s
domain: 经济
---

# %s —— 经济案 canon（T-0314 裁定生成件）

kind=%s ref=%s

## 案由
%s

## 裁定回改点
（决策席裁定后人工补录——翻译器 gatefx 机械类词径或语义条目径。）
`, slug, kind, ref, slug, kind, ref, nums)
	if err := os.WriteFile(dir+"/SKILL.md", []byte(doc), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("SKILL 件落 %s/SKILL.md——go run ./tools/kbgen skills 重建索引\n", dir)
}
