#!/usr/bin/env bash
# check_queue_birth.sh —— 判断型消费面出生闸（工单O-35484①——evalfloor
# 幻影 623h 隐身/同根簇 30+ 火复发/digest finding 派生缺，同型病三得
# 打地鼠根因=出生无闸）。凡新增判断型消费面（视图/队列表/消费谓词）
# 须配三件套——①保底谓词（积压龄界，死角可见）②折叠签名（同根去重）
# ③派生规则（finding→票），缺一闸红。
#
# 登记表=docs/queue_birth_registry.txt（面名|类|保底谓词|折叠签名|派生
# 规则|豁免，# 注释/空行免）。机扫面=migrations/*.sql 全部
# CREATE [OR REPLACE] VIEW 名；队列表/消费谓词等机扫不可名状面由登记表
# 钦定。存量面普查豁免落档（只挡新生不追旧账）。
# 桩测面：QUEUE_BIRTH_MIG_DIR/QUEUE_BIRTH_REG 可指夹具树
# （check_queue_birth_test.sh 喂未登记/缺件/幻影三案）。
set -uo pipefail
if [ -n "${QUEUE_BIRTH_SCAN_ROOT:-}" ]; then
	ROOT=$QUEUE_BIRTH_SCAN_ROOT
else
	ROOT=$(git rev-parse --show-toplevel)
fi
MIGDIR="${QUEUE_BIRTH_MIG_DIR:-$ROOT/migrations}"
REG="${QUEUE_BIRTH_REG:-$ROOT/docs/queue_birth_registry.txt}"
[ -f "$REG" ] || { echo "❌ [queue-birth] $REG 缺席——消费面出生登记表未立" >&2; exit 1; }
[ -d "$MIGDIR" ] || { echo "❌ [queue-birth] $MIGDIR 缺席——迁移目录不可达" >&2; exit 1; }

# ── 机扫：迁移内全部 view 名（CREATE [OR REPLACE] VIEW <name> 去重）──
declare -A SEENV=()
while IFS= read -r v; do
	[ -n "$v" ] && SEENV[$v]=1
done < <(grep -rhoE 'CREATE[[:space:]]+(OR[[:space:]]+REPLACE[[:space:]]+)?VIEW[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*' "$MIGDIR"/*.sql 2>/dev/null \
	| awk '{print $NF}' | sort -u)

# ── 登记表解析 ─────────────────────────────────────────────────
declare -A REGROW=()   # 面名 → 整行（判存量/三件套）
declare -A REGCLS=()
FAIL=0
while IFS= read -r line; do
	line=${line%$'\r'}
	case "$line" in ''|'#'*) continue;; esac
	nf=$(printf '%s' "$line" | awk -F'|' '{print NF}')
	if [ "$nf" -ne 6 ]; then
		echo "❌ [queue-birth] 登记行字段数=$nf≠6: $line" >&2; FAIL=1; continue
	fi
	IFS='|' read -r name cls floor fold derive exempt <<<"$line"
	name=$(printf '%s' "$name" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	cls=$(printf '%s' "$cls" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	floor=$(printf '%s' "$floor" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	fold=$(printf '%s' "$fold" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	derive=$(printf '%s' "$derive" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	exempt=$(printf '%s' "$exempt" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	[ -n "$name" ] || { echo "❌ [queue-birth] 登记行面名空: $line" >&2; FAIL=1; continue; }
	case "$cls" in view|queue|谓词) ;; *)
		echo "❌ [queue-birth] $name 类不符=$cls（册=view|queue|谓词）" >&2; FAIL=1; continue;; esac
	# 存量豁免行：豁免列必须「存量」起头（防新生滥挂豁免——词面闸）
	if [ -n "$exempt" ]; then
		case "$exempt" in 存量*) ;; *)
			echo "❌ [queue-birth] $name 豁免词不符（只认「存量…」——新生禁挂豁免）: $exempt" >&2
			FAIL=1; continue;; esac
	else
		# 非豁免行=新生面——三件套缺一「-」/空即红
		miss=""
		for kv in "保底谓词|$floor" "折叠签名|$fold" "派生规则|$derive"; do
			k=${kv%%|*}; v=${kv#*|}
			if [ -z "$v" ] || [ "$v" = "-" ]; then miss="$miss $k"; fi
		done
		if [ -n "$miss" ]; then
			echo "❌ [queue-birth] $name 缺三件套:$miss——新增判断型消费面须配齐（保底谓词/折叠签名/派生规则）" >&2
			FAIL=1; continue
		fi
	fi
	REGROW[$name]="$line"
	REGCLS[$name]=$cls
done < "$REG"

# ── 对拍①：机扫 view ⊆ 登记（未登记=新增面未配三件套）───────────
for v in "${!SEENV[@]}"; do
	[ -n "${REGROW[$v]:-}" ] || {
		echo "❌ [queue-birth] 消费面 $v 未登记——新增判断型消费面须入 docs/queue_birth_registry.txt 配三件套（保底谓词/折叠签名/派生规则）" >&2
		FAIL=1
	}
done

# ── 对拍②：登记 view 行 ⊆ 机扫集（幻影登记=闸红）────────────────
for n in "${!REGROW[@]}"; do
	if [ "${REGCLS[$n]}" = "view" ] && [ -z "${SEENV[$n]:-}" ]; then
		echo "❌ [queue-birth] 登记面 $n（类=view）机扫集无此 view——幻影登记或已删未除名" >&2
		FAIL=1
	fi
done

if [ "$FAIL" -ne 0 ]; then
	echo "❌ [queue-birth] 消费面出生闸红——判据见上（工单O-35484①）" >&2
	exit 1
fi
echo "OK: check_queue_birth（消费面机扫 ${#SEENV[@]} view 全登记——存量豁免在案/新生面三件套齐）"
