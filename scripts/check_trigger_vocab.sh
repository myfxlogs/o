#!/usr/bin/env bash
# check_trigger_vocab.sh —— 结案触发词谱×结案认知集迁移覆盖对拍闸
# （O-41388 病灶钉：dirtrig 词族码面产词后认知集迁移未补——
# fin_guard_close.trigger_kind 新词=postmortem 全打 trigger_unknown
# 误标烧诊断额度，2026-11-28 dir_* 族 18 张实证）。
# 断言：码面产词面 ⊆ migrations 对 guard.postmortem.known_triggers
# 覆盖集（种子行或幂等追加皆可——词出现在提及该键的迁移文件即盖）。
set -u
cd "$(dirname "$0")/.."

# 码面词谱三源：
#   closeTrig{kind:"X"} 字面量（decide/dirtrig/emergency_exit 全产点）
#   dirTrigKind/ddTrigger 常量值（dirline/ddline 留痕词）
#   postmortem 缺省词 manual（两源皆缺=手平）
LITS=$(grep -rhoE 'closeTrig\{kind:\s*"[a-z_0-9]+"' internal/ --include='*.go' \
  | grep -oE '"[a-z_0-9]+"' | tr -d '"')
CONSTS=$(grep -rhoE '(dirTrigKind|ddTrigger)\s*=\s*"[a-z_0-9]+"' internal/ \
  --include='*.go' | grep -oE '"[a-z_0-9]+"' | tr -d '"')
KINDS=$(printf '%s\n%s\nmanual\n' "$LITS" "$CONSTS" | sort -u)

MIGS=$(grep -rl 'known_triggers' migrations/ 2>/dev/null || true) # tolerate: grep 零匹配rc=1=合法空集径——下行显式判零文件即红，非吞败
if [ -z "$MIGS" ]; then
  echo "RED[trigger-vocab] 认知集迁移面零文件" >&2
  exit 1
fi

fail=0
for k in $KINDS; do
  hit=$(echo "$MIGS" | xargs grep -l "\"$k\"" 2>/dev/null | head -1)
  if [ -z "$hit" ]; then
    echo "RED[trigger-vocab] trigger 词 '$k' 无迁移覆盖——码面产词未入认知集" >&2
    fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "OK[trigger-vocab] $(echo "$KINDS" | wc -l | tr -d ' ') 词全盖"
exit "$fail"
