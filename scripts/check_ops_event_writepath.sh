#!/usr/bin/env bash
# check_ops_event_writepath.sh —— 升件写径归一白名单闸（工单O-30514
# 形散神不散根修①）：仓内 .sh 面 esc 写径只许出现在白名单列件，
# 违例=闸红。escalate.sh/deploy_watch.sh 归一库内函件 ops_esc()
# （migrations/fin_0228——note=prefix——detail、24h 前缀同窗去重、
# psql -v 参数化喂参引号结构性绝根）后不入册=直写回归即被本闸咬住。
#
# 判据（两判形，防「恢复/量表」行误咬——dispatch.sh INSERT...SELECT
# 读 e.esc_scope、note 载 '升件#id' 销件行皆非 esc 写；本件注释面
# 一律避写字面判式防自咬）：
#   A 同行列名形：直写 ops_event 且列单含 esc_class|esc_scope——
#     esc 元数据列只服务升件行，列单出现=esc 写；
#   B 事件字面量形：INSERT 窗口（起行+下5行）含 逗号引号升件引号
#     ——event 位字面量；'升件#'（销件引用）/equals升件（WHERE 谓词）
#     不匹配（闭合引号须紧随「件」）。
# 非 esc 写（心跳/部署/唤起/等锁/恢复/量表等 ops_event 说明类行）
# 非本闸面——写径归一只钉升件族。白名单=scripts/ops_event_esc_
# whitelist.txt（# 注释+仓根相对路径行；缺席=fail-closed）。
set -uo pipefail
# OPS_WRITEPATH_ROOT=夹具根覆盖（桩测注入口——缺省=git 仓根）。
ROOT=${OPS_WRITEPATH_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null)}
[ -n "$ROOT" ] && cd "$ROOT" || { echo "❌ [ops-writepath] 仓根不可解——fail-closed"; exit 1; }
WL=scripts/ops_event_esc_whitelist.txt
[ -f "$WL" ] || { echo "❌ [ops-writepath] 白名单缺席 $WL——判据失载 fail-closed"; exit 1; }
mapfile -t WLPATHS < <(grep -vE '^\s*(#|$)' "$WL")
[ ${#WLPATHS[@]} -gt 0 ] || { echo "❌ [ops-writepath] 白名单空集——判据失载 fail-closed"; exit 1; }

bad=""
while IFS= read -r f; do
	rel=${f#./}
	if printf '%s\n' "${WLPATHS[@]}" | grep -qxF "$rel"; then continue; fi
	# A 形：列单含 esc 元数据列（同行判——列单恒与 INSERT 同行）
	while IFS= read -r hit; do
		[ -n "$hit" ] && bad="$bad$rel:${hit%%:*} (列单 esc_class/esc_scope)"$'\n'
	done < <(grep -nE "INSERT INTO ops_event\([^)]*esc_(class|scope)" "$f" 2>/dev/null | cut -d: -f1 || true) # tolerate: grep 零命中 rc1=正常空面——while 读空即零违例
	# B 形：窗口内 event='升件' 字面量
	while IFS= read -r ln; do
		[ -z "$ln" ] && continue
		win=$(sed -n "${ln},$((ln+5))p" "$f")
		printf '%s' "$win" | grep -qE ",[[:space:]]*'升件'" \
			&& bad="$bad$rel:$ln (event='升件' 字面量)"$'\n'
	done < <(grep -nE 'INSERT INTO ops_even[t]' "$f" 2>/dev/null | cut -d: -f1 || true) # tolerate: grep 零命中 rc1=正常空面——while 读空即零违例
done < <(find . -name '*.sh' -type f -not -path './.git/*' | sort)

if [ -n "$bad" ]; then
	printf '%s' "$bad" | while IFS= read -r b; do
		[ -n "$b" ] && echo "❌ [ops-writepath] $b —— 升件写径归一 ops_esc()（fin_0228），白名单外直写=闸红"
	done
	echo "❌ [ops-writepath] 升件写径白名单闸红——新增 esc 写径须先论证不能走函件再入 scripts/ops_event_esc_whitelist.txt 注事由" >&2
	exit 1
fi
echo "OK: check_ops_event_writepath（.sh 面 esc 写径全在册——escalate.sh/deploy_watch.sh 函件径零直写）"
