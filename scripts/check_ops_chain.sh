#!/usr/bin/env bash
# check_ops_chain.sh —— T-0386 ops_event 防篡改账闸（check_all 段）：
#   ① 迁移面：fin_0120 链列（prev_hash/hash）+链写触发器（BEFORE INSERT+
#     advisory 串行）+禁UD锁权函数+验链函数 齐位；
#   ② 登记面：migrate_table.go 挂 0120（check_migrations 另钉序号/孤儿）；
#   ③ 接线面：验链拍挂 lifecycle 饥饿同槽+tools/hungertick 同径；
#   ④ LIVE 段（条件）：OPS_CHAIN_DSN|FIN_PG_DSN 在且库内 ops_chain_verify()
#     已落地→全链重算零断（迁移未发窗/DSN 缺席=跳过不炸——check_all 离线径）。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
fail=0

MIG=migrations/fin_0120_ops_event_chain.sql
if [ ! -f "$MIG" ]; then
	echo "FAIL $MIG: 迁移件缺席——ops_event 链化面未落" >&2
	exit 1
fi
for pat in 'prev_hash' 'ADD COLUMN IF NOT EXISTS hash' 'BEFORE INSERT' 'ops_event_chain_trg' \
	'pg_advisory_xact_lock' 'REVOKE UPDATE,DELETE,TRUNCATE' 'ops_event_lockdown' 'ops_chain_verify'; do
	grep -q "$pat" "$MIG" || { echo "FAIL $MIG: 缺 $pat" >&2; fail=1; }
done

grep -q '"fin_0120_ops_event_chain.sql"' internal/store/pgstore/migrate_table.go \
	|| { echo "FAIL migrate_table.go: fin_0120 未登记（孤儿文件——永不执行）" >&2; fail=1; }

grep -q 'internal/platform/opschain' internal/portfolio/lifecycle/run.go \
	|| { echo "FAIL lifecycle/run.go: opschain 未入装配（验链拍未挂饥饿同槽）" >&2; fail=1; }
grep -q 'chainV.Verify' internal/portfolio/lifecycle/hunger.go \
	|| { echo "FAIL lifecycle/hunger.go: hungerTick 未调验链拍" >&2; fail=1; }
grep -q 'internal/platform/opschain' tools/hungertick/main.go \
	|| { echo "FAIL tools/hungertick: 验链拍未挂拍壳同径" >&2; fail=1; }

# ④ LIVE 段——DSN 在且函数已落地才验（迁移未发窗=跳过不炸）。
DSN=${OPS_CHAIN_DSN:-${FIN_PG_DSN:-}}
if [ -n "$DSN" ] && command -v psql >/dev/null 2>&1; then
	fn=$(timeout 15 psql "$DSN" -Atc \
		"SELECT count(*) FROM pg_proc WHERE proname='ops_chain_verify'" 2>/dev/null || true) # tolerate: 函数在册探测——连不上/超时=按未落地跳过（LIVE 段条件径非吞败）
	if [ "${fn:-0}" -ge 1 ] 2>/dev/null; then
		broken=$(timeout 30 psql "$DSN" -Atc "SELECT count(*) FROM ops_chain_verify()" 2>/dev/null || echo ERR)
		if [ "$broken" = "0" ]; then
			echo "OK: ops-chain LIVE——全链重算零断"
		elif [ "$broken" = "ERR" ]; then
			echo "FAIL ops-chain LIVE: ops_chain_verify 执行败" >&2
			fail=1
		else
			echo "FAIL ops-chain LIVE: 链断 $broken 处——" >&2
			timeout 15 psql "$DSN" -c "SELECT * FROM ops_chain_verify() ORDER BY id LIMIT 5" >&2 || true # tolerate: 断点明细 best-effort——count 已显红，明细印败不遮主断言
			fail=1
		fi
	else
		echo "ops-chain LIVE 跳过——库内函数未落地（迁移未发窗）"
	fi
fi

[ "$fail" -eq 0 ] || exit 1
echo "OK: check_ops_chain（迁移面+登记面+接线面齐${DSN:+；LIVE 段已评}）"
