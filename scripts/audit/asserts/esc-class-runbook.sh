#!/usr/bin/env bash
# 审计/探测面 RED 锚（audit/1113-probe2）：仓内全部升件 esc_class 字面量
# 须在 runbook.md 有判词档节（「- **<类>**」节头）。
# 产线实证：esc_class='读败' 由 venue_fail_gate/macro_remind/calendar 三产者
# 实发（mt5×3 @11-09）而 runbook 判词档零节（行199仅处置例句内嵌词非档）;
# esc_runbook_align_test 只检 cmd/core escEdgeU scope 名,class 层+仓外产者
# 不在机检=漂移无闸。正确语义=每个实发 esc_class 有档节。修好断言自然绿。
set -u
cd "${1:-/opt/o-wt/audit-1113-probe2}"
RB=scripts/cognition/runbook.md
MISS=""
CLASSES=$(
  # ① escEdgeU 第3参类字面量（cmd/core）
  grep -rhoE 'escEdgeUV\(ctx, +deadSince, +"[^"]+"' cmd/core/*.go \
    | sed -E 's/.*"([^"]+)"$/\1/'
  # ② 类常量字面量（EscClass/ClassXxx/probeEscClass——仓外产者同构）
  grep -rhoE '(EscClass|escClass|probeEscClass|Class[A-Za-z]*)[[:space:]]*=[[:space:]]*"[^"]+"' internal/ cmd/ --include='*.go' \
    | sed -E 's/.*"([^"]+)"/\1/'
  # ③ VALUES 位类字面量（参数化写入的类——'读败'/'venue漂移'/... CJK 词）
  grep -rhoE "'(读败|写败|venue漂移|接入败|谱域缺行|谱域不可判|执行径演习|账链断|观测盲|守卫环|改进提案|对账|断流|资金闸|商业产出|孤儿账|唤知|数据缺|能力债|预期产出|审计)'" internal/ cmd/ --include='*.go' \
    | tr -d "'"
) 
for c in $(printf '%s\n' "$CLASSES" | sort -u | grep -v '^$'); do
  grep -qE "^[[:space:]]*- \*\*[^*]*${c}[^*]*\*\*" "$RB" || MISS="$MISS $c"
done
if [ -n "$MISS" ]; then
  echo "assert RED: 实发 esc_class 缺 runbook「- **类**」判词档节:$MISS"
  exit 1
fi
echo "assert GREEN: 全量 esc_class 字面量有档节"
