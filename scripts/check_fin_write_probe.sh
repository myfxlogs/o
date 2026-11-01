#!/usr/bin/env bash
# check_fin_write_probe.sh —— T-0232 验收锚①：OCI 侧四角色对 fin 钱径态表
# BEGIN;INSERT/UPDATE;ROLLBACK 探针必须失败（research_ro/review_rw/roster_rw/
# collector_rw；projector_rw 只验 fin_position/fin_strategy 读不收写——
# fin_proposal 镜像 upsert 是其钦定面故不在探针表列）。
# 用法：scripts/check_fin_write_probe.sh（SG_HOST 可覆盖；SSH+docker exec 径
# 以 o 超户 SET ROLE 模拟——零凭据入脚本）。
set -euo pipefail
SG_HOST="${SG_HOST:-root@192.229.85.35}"
q() { timeout 20 ssh -n -o BatchMode=yes -o ConnectTimeout=8 "$SG_HOST" \
	"docker exec o-postgres psql -U o -d o -c \"$1\"" 2>&1 || true; }
fail=0
for role in research_ro review_rw roster_rw collector_rw; do
	for tbl in fin_position fin_strategy fin_proposal; do
		out=$(q "SET ROLE $role; INSERT INTO $tbl(id) VALUES(-1);")
		if echo "$out" | grep -qiE 'permission denied|INSERT 0 1'; then :; fi
		if echo "$out" | grep -q 'INSERT 0 1'; then
			echo "FAIL[fin-probe] $role INSERT $tbl 成功——收窄破"; fail=1
		elif ! echo "$out" | grep -qi 'permission denied'; then
			echo "FAIL[fin-probe] $role INSERT $tbl 非权限拒（可疑通过）: $out"; fail=1
		fi
		out=$(q "SET ROLE $role; UPDATE $tbl SET id=id WHERE false;")
		if ! echo "$out" | grep -qi 'permission denied'; then
			echo "FAIL[fin-probe] $role UPDATE $tbl 未被权限拒: $out"; fail=1
		fi
	done
done
# review_rw 判决事实追加=合法存续面（乙案——反向钉：可写防误收）。
out=$(q "SET ROLE review_rw; INSERT INTO fin_review(position_ref,verdict,detail) VALUES(-1,'marginal','{}');")
echo "$out" | grep -q 'INSERT 0 1' || { echo "FAIL[fin-probe] review_rw INSERT fin_review 拒——判决道断: $out"; fail=1; }
q "DELETE FROM fin_review WHERE position_ref=-1" >/dev/null
out=$(q "SET ROLE roster_rw; INSERT INTO ops_event(seat,event,ticket) VALUES('probe','probe','probe');")
echo "$out" | grep -q 'INSERT 0 1' || { echo "FAIL[fin-probe] roster_rw INSERT ops_event 拒——自报道断: $out"; fail=1; }
q "DELETE FROM ops_event WHERE seat='probe'" >/dev/null
[ "$fail" = 0 ] && echo "OK[fin-probe] 四角色×fin_position/fin_strategy/fin_proposal 写探针全拒+判决/自报合法面通" || exit 1
