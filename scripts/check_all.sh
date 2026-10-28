#!/usr/bin/env bash
# check_all.sh — CI 同链全检（业主令 2026-10-27：绿了才推）——build→vet→
# import→gofmt→env→rejects→migrations→go test 全谱，绿后盖鲜戳
# .git/check-green-<SHA>（pre-push 验戳放行；无戳/陈戳=拒）。
# worktree 缺 gen/dist 件先跑 scripts/provision_worktree.sh。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"
GITDIR=$(git rev-parse --git-dir)
STAMP="$GITDIR/check-green-$(git rev-parse HEAD)"
rm -f "$STAMP"   # 重跑先毁旧戳——中途败不留残戳

fail() { echo "❌ [check-all] $1——绿了才推" >&2; exit 1; }
step() { echo "── $1" >&2; }

step "go build";    go build ./...                              || fail "go build 红"
step "go vet";      go vet ./...                                || fail "go vet 红"
step "import 方向"; bash scripts/check_import_direction.sh      || fail "import 门禁红"
step "gofmt"
UNFMT=$(gofmt -l . 2>/dev/null || true)
[ -z "$UNFMT" ] || fail "gofmt 未格式件: $UNFMT"
step "env-reads";   bash scripts/check-env-reads.sh             || fail "env 门禁红"
step "rejects";     bash scripts/check_rejects_keys.sh          || fail "rejects 键红"
step "migrations";  bash scripts/check_migrations.sh            || fail "迁移机检红"
step "go test 全谱"
if [ -z "${O_TEST_PG_DSN:-}" ] &&
   docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^o-postgres-test$'; then
  export O_TEST_PG_DSN="postgres://o:o@localhost:5434/o_test?sslmode=disable"
fi
# T-0212：去 -count=1 放行 Go 测试缓存——同 tree 复测近零本（业主令：全量
# uncached 25-40min/次太贵）。缓存键=文件+依赖指纹非 git 态——外部依赖型
# （真实 PG/venue）陈判窗口钦定放行（CI 服务端仍全谱 uncached 兜真伪）。
go test ./... -timeout=1800s                                  || fail "go test 红"

date -Is > "$STAMP"
echo "✅ [check-all] 全链绿——戳落 $STAMP" >&2
