#!/usr/bin/env bash
# check_resident_units.sh —— 常驻件机读登记对拍闸（工单O-30515——器官增生
# 无卡口根修②：AGENTS.md「常驻盒内件」行是事实登记但无机械闸，新常驻件
# 长出无对拍）。登记表=docs/resident_units.txt（件名|类|拍距|宿点|职责），
# 实扫三面——
#   cmd/*/main.go            入口集      → 类=cmd入口
#   deploy/**/*.service      单元件      → 类=oci单元
#   deploy oneshot ExecStart 脚本 payload → 类=定时脚本
# 未登记∨登记件已删∨类不符=闸红；拍内 beat/probe/face 细胞层不查（cells 非器官）。
# 桩测面：RESIDENT_SCAN_ROOT/RESIDENT_UNITS_FILE 可指夹具树
# （check_resident_units_test.sh 喂未登记/已删双红案）。
set -uo pipefail
if [ -n "${RESIDENT_SCAN_ROOT:-}" ]; then
	ROOT=$RESIDENT_SCAN_ROOT
else
	ROOT=$(git rev-parse --show-toplevel)
fi
REG="${RESIDENT_UNITS_FILE:-$ROOT/docs/resident_units.txt}"
[ -f "$REG" ] || { echo "❌ [resident-units] $REG 缺席——常驻件登记表未立" >&2; exit 1; }

# ── 登记表解析：件名|类|拍距|宿点|职责（# 注释/空行免）────────────────
declare -A REGCLS=()
while IFS= read -r line; do
	line=${line%$'\r'}
	case "$line" in ''|'#'*) continue;; esac
	nf=$(printf '%s' "$line" | awk -F'|' '{print NF}')
	[ "$nf" -eq 5 ] || { echo "❌ [resident-units] 登记行字段数=$nf≠5: $line" >&2; exit 1; }
	IFS='|' read -r name cls pace host duty <<<"$line"
	name=$(printf '%s' "$name" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	cls=$(printf '%s' "$cls" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	[ -n "$name" ] || continue
	case "$cls" in cmd入口|oci单元|定时脚本) ;; *)
		echo "❌ [resident-units] 未知类 '$cls'（件=$name）——类册=cmd入口|oci单元|定时脚本" >&2; exit 1;; esac
	[ -z "${REGCLS[$name]:-}" ] || { echo "❌ [resident-units] 登记件名重复: $name" >&2; exit 1; }
	REGCLS[$name]=$cls
done < "$REG"
[ "${#REGCLS[@]}" -gt 0 ] || { echo "❌ [resident-units] 登记表空册——零登记=闸无对象 fail-closed" >&2; exit 1; }

# ── 实扫面（扫的就是表口径——三集互不混）────────────────────────────
declare -A WANT=()   # key → 期望类
# ① cmd/*/main.go 入口集
if [ -d "$ROOT/cmd" ]; then
	while IFS= read -r f; do
		rel=${f#"$ROOT"/cmd/}; rel=${rel%/main.go}
		[ -n "$rel" ] && WANT["cmd/$rel"]=cmd入口
	done < <(find "$ROOT/cmd" -mindepth 2 -maxdepth 2 -name main.go | sort)
fi
# ② deploy/**/*.service 单元件（.timer/.path/.slice=调度/资源壳不单列）
if [ -d "$ROOT/deploy" ]; then
	while IFS= read -r f; do
		rel=${f#"$ROOT"/deploy/}; rel=${rel%.service}
		[ -n "$rel" ] && WANT["$rel"]=oci单元
	done < <(find "$ROOT/deploy" -name '*.service' | sort)
	# ③ deploy oneshot ExecStart 脚本 payload（长驻件 ExecStart 不收——
	#    隧道/服务壳归 oci单元行，脚本 payload 只认定时 oneshot 面）。
	#    归一=剥 /opt/o/、./ 前缀后须 scripts/*.sh 起——/opt/meta 等仓外
	#    scripts 径不收（非本仓常驻件）。
	SCRIPTS=$(find "$ROOT/deploy" -name '*.service' | sort | while IFS= read -r f; do
		grep -q '^Type=oneshot' "$f" || continue
		grep -E '^ExecStart=' "$f" | sed 's/^ExecStart=//' \
			| tr ' \t' '\n\n' | tr -d "\"'" \
			| while IFS= read -r tok; do
				case "$tok" in
					/opt/o/*) tok=${tok#/opt/o/} ;;
					./*)     tok=${tok#./} ;;
				esac
				case "$tok" in scripts/*.sh) printf '%s\n' "$tok" ;; esac
			done
	done | sort -u)
	while IFS= read -r s; do [ -n "$s" ] && WANT["$s"]=定时脚本; done <<< "$SCRIPTS"
fi

# ── 双向对拍：扫件未登记 / 登记件已删 / 类不符 ────────────────────────
fail=0
for k in "${!WANT[@]}"; do
	if [ -z "${REGCLS[$k]:-}" ]; then
		echo "❌ [resident-units] 未登记常驻件: $k（类应=${WANT[$k]}）——登记表补行" >&2; fail=1
	elif [ "${REGCLS[$k]}" != "${WANT[$k]}" ]; then
		echo "❌ [resident-units] 类不符: $k 登记=${REGCLS[$k]} 实扫=${WANT[$k]}" >&2; fail=1
	fi
done
for k in "${!REGCLS[@]}"; do
	if [ -z "${WANT[$k]:-}" ]; then
		echo "❌ [resident-units] 登记件已删/不可判: $k（类=${REGCLS[$k]}）——登记表销行或复件" >&2; fail=1
		continue
	fi
	# 定时脚本 payload 文件本体须在仓（单元在册而脚本已删=拍壳空转）
	if [ "${REGCLS[$k]}" = 定时脚本 ] && [ ! -f "$ROOT/$k" ]; then
		echo "❌ [resident-units] 定时脚本 payload 缺席: $k——单元 ExecStart 指空件" >&2; fail=1
	fi
done
[ "$fail" = 0 ] || exit 1
echo "✅ [resident-units] 对拍绿——登记 ${#REGCLS[@]} 件 ↔ 实扫三集齐（cmd入口/oci单元/定时脚本）" >&2
exit 0
