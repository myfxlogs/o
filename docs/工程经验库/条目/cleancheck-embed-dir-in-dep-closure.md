---
schema_version: 2
id: cleancheck-embed-dir-in-dep-closure
kind: pitfall
risk: medium
status: verified
keywords: [证空判面, go:embed, dep闭包, deploy_watch, image_face_clean, arm_provably_clean, 迁移饿死, 破口审计]
scope: [scripts/deploy_watch.sh, scripts/arm_faces_lib.sh, deploy/deploy-faces.tsv, o/migrations, cmd/core]
summary: 证空/谱系判面按目录语义猜归属会误判——migrations/ 是 go:embed 包，在 cmd/core dep 闭包内，迁移件必判证内；破口判词须 go list -deps 实算+账证，勿按「数据目录」想象
first_seen: 2026-11-20
last_verified: 2026-11-20
---

# 证空判面审计：embed 目录在 dep 闭包内，不是「纯数据面」

**坑形**：工单#25802 破口前提（前序审计判词）称 `migrations/*.sql`
是纯数据文件、天然落在 `go list -deps ./cmd/core` 闭包外，故迁移-only
commit 判证空→零重启→Migrate 永等启动点饿死。实测证伪：`o/migrations`
是 Go embed 包（`//go:embed *.sql domain.tsv`），在 cmd/core 闭包内
（dirs 第 4 位）；且真实迁移 commit 恒同碰
`internal/store/pgstore/migrate_table.go`（亦闭包内）——双保险下迁移件
必判证内，饿死径不存在。

**判别**：`image_face_clean` / `arm_provably_clean` 的判据是「路径 ∈
dep 闭包目录集」——闭包按**包**计，不按文件后缀；任何 `//go:embed`
宿主目录的内容物（.sql/.tsv/模板/静态件）随包入闭包。判面前先实算
`go list -f '{{.Dir}}' -deps ./cmd/<件>`，勿按目录名/后缀推断归属。

**修法**：破口判词三锚——①闭包实算（dirs 含 migrations）；②账证
（含迁移的 merge `45591b4f`/`6ecbac469` 后 deploy/deploy-oci rc=0
齐落）；③臂级判词按各臂实测闭包非对称照搬（collecto 无
`o/migrations` 依赖、零 Migrate 调用，其 `migrations/*` 判证外=正确）。

**推广**：凡「X 类文件必被证空跳过」的破口结论，先实算闭包再立案——
embed/生成码/资产目录常是包成员；证空判面审计的证据级 = go list
输出 + 账证行，不是目录语义想象。
