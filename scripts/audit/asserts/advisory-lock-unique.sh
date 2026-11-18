#!/usr/bin/env bash
# 审计/唤知环 RED 锚（audit/1117-cogring3）：各 dispatch 臂 PG advisory
# lock 键须全册唯一——键册钦定 7969=cognition/7970=audit/7971=self；
# dispatch_worker.sh 复用 7970 与 audit dispatch 撞键——跨道临界区
# 非意串行（唤知唤起拍被审计唤起挡/反之），同型槽位令分工件该独立
# 槽不互拦反成共享闸。正确语义=每臂 lock 键册内唯一；修好断言自然绿。
set -u
cd "$(dirname "$0")/../../.." || exit 2
declare -A OWN
DUP=0
for f in scripts/audit/dispatch.sh scripts/cognition/dispatch.sh \
	scripts/cognition/dispatch_self.sh scripts/cognition/dispatch_worker.sh; do
	[ -f "$f" ] || continue
	key=$(grep -oE 'pg_advisory_lock\([0-9]+\)' "$f" | head -1 | grep -oE '[0-9]+')
	[ -n "$key" ] || continue
	if [ -n "${OWN[$key]:-}" ]; then
		echo "assert RED: advisory lock 键 $key 双主——$f 与 ${OWN[$key]} 撞册（跨道非意串行=槽位令不互拦被反吃）"
		DUP=1
	else
		OWN[$key]=$f
	fi
done
[ "$DUP" = 0 ] && { echo "assert GREEN: dispatch 锁键册唯一（${!OWN[@]}）"; exit 0; }
exit 1
