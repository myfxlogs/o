#!/usr/bin/env bash
# 票号唯一性机检（T-0163）——双席并行立单撞号根治（T-0154/T-0160 双撞实证）。
#
# 用法：
#   check_ticket_numbers.sh            # 存量查重：全仓（index）票号扫一遍，
#                                      # 任一号多于一票即逐对红出（exit 1=存在共存）
#   check_ticket_numbers.sh --staged   # pre-commit 增量闸：staged 新增/拷贝/改名票
#                                      # 逐号核对，同号已存即红报冲突件名（exit 1）；
#                                      # 新立票过闸后附取号提示（本地视野≠远端为准）
#
# 计数源=git index（本 commit 落定后的全仓票面：staged 删票不计、staged 加票计），
# 不取工作树 glob——后者会把 staged 删票误计入数。
# 存量双号（T-0160 两张）决策裁注记共存、不溯及：--staged 只拦增量（M 改票不触发），
# 存量清单由无参模式出具。
set -euo pipefail

TICKET_RE='^docs/tasks/T-[0-9]{4,}-'

num_of() { sed -E 's|^docs/tasks/T-([0-9]{4,})-.*|\1|'; }
num_files() { # $1=票号 → index 内该号票文件（每行一个）
  git -c core.quotepath=off ls-files -- "docs/tasks/T-$1-*.md" "docs/tasks/T-$1.md"
}

if [ "${1:-}" = "--staged" ]; then
  # staged 新路径：A/C 单路径、R 双路径取新（$NF）；M/D 不在滤内——存量票改注不触发本闸
  new_paths=$(git -c core.quotepath=off diff --cached --name-status --diff-filter=ACR \
    | awk -F'\t' '{print $NF}' | grep -E "$TICKET_RE" || true)
  if [ -z "$new_paths" ]; then
    exit 0
  fi

  status=0
  while IFS= read -r n; do
    dups=$(num_files "$n")
    cnt=$(printf '%s\n' "$dups" | grep -c . || true)
    if [ "$cnt" -gt 1 ]; then
      echo "FAIL(T-0163): 票号 T-$n 撞号——index 内同号票 $cnt 张（双席取号竞态；后到让号重编，或挂起交决策裁）：" >&2
      printf '%s\n' "$dups" | sed 's/^/  /' >&2
      status=1
    fi
  done <<< "$(printf '%s\n' "$new_paths" | num_of | sort -u)"
  if [ "$status" -eq 1 ]; then
    exit 1
  fi

  # 取号推进提示（不拦）：本地最大号+1 非必对——双席并行远端为准（T-0163 票面）
  lmax=$(git -c core.quotepath=off ls-files -- 'docs/tasks/T-*.md' \
    | grep -oE 'T-[0-9]{4,}' | sed 's/^T-//' | sort -n | tail -1 || true)
  echo "NOTE(T-0163): 新立票过本地唯一闸；本地最大票号=T-$lmax。取号非必对（双席视野差=撞号根因）——push 前 git fetch origin 并核 origin/master:docs/tasks/ 有无同号/更高号（远端为准）。" >&2
  exit 0
fi

# 无参：存量查重——exit 1=存在同号共存；存量处置归决策，本模式只出清单不拦入链
dup_nums=$(git -c core.quotepath=off ls-files -- 'docs/tasks/T-*.md' \
  | grep -E "$TICKET_RE" | num_of | sort | uniq -d || true)
status=0
while IFS= read -r n; do
  if [ -z "$n" ]; then
    continue
  fi
  cnt=$(num_files "$n" | grep -c . || true)
  echo "T-$n 同号共存 ${cnt} 张：" >&2
  num_files "$n" | sed 's/^/  /' >&2
  status=1
done <<< "$dup_nums"
if [ "$status" -eq 0 ]; then
  echo "OK: 票号全仓唯一" >&2
fi
exit "$status"
