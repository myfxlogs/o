#!/usr/bin/env bash
# 票号唯一性机检（T-0163）——双席并行立单撞号根治（T-0154/T-0160 双撞实证）。
#
# 用法：
#   check_ticket_numbers.sh            # 存量查重：全仓（index）票号扫一遍，
#                                      # 任一号多于一票即逐对红出（exit 1=存在共存）
#   check_ticket_numbers.sh --staged   # pre-commit 增量闸：staged 新增/拷贝/改名票
#                                      # 逐号核对，同号已存即红报冲突件名（exit 1）；
#                                      # 新立票过闸后附取号提示（本地视野≠远端为准）
#   check_ticket_numbers.sh --remote   # pre-push 远端对账（T-0219）：fetch 后扫
#                                      # origin/master:docs/tasks/——新立票号远端已存
#                                      # 或 ≤远端最大号即硬红；fetch 败降级 WARN 不拦
#
# 计数源=git index（本 commit 落定后的全仓票面：staged 删票不计、staged 加票计），
# 不取工作树 glob——后者会把 staged 删票误计入数。
# 存量双号（T-0160 两张）决策裁注记共存、不溯及：--staged 只拦增量（M 改票不触发），
# 存量清单由无参模式出具。
set -euo pipefail

TICKET_RE='^docs/tasks/(done/)?T-[0-9]{4,}[[:alpha:]]?(-|\.md$)'

ticket_key() { sed -E 's#^docs/tasks/(done/)?(T-[0-9]{4,}[[:alpha:]]?)(-|\.md$).*#\2#'; }
ticket_num() { sed -E 's#^docs/tasks/(done/)?T-([0-9]{4,})[[:alpha:]]?(-|\.md$).*#\2#'; }
ticket_files() { # $1=路径树，返回树根与完成归档下全部票
  git -c core.quotepath=off ls-files -- "$1/T-*.md" "$1/done/T-*.md"
}
num_files() { # $1=票号键（含可选字母后缀）→ index 内该票文件（每行一个）
  git -c core.quotepath=off ls-files -- \
    "docs/tasks/$1-*.md" "docs/tasks/$1.md" \
    "docs/tasks/done/$1-*.md" "docs/tasks/done/$1.md"
}
staged_archive_move() { # 同内容根目录→done/改名不是新增票，不重审存量重号
  local path=$1
  [[ "$path" == docs/tasks/done/T-* ]] || return 1
  local old
  old=$(git diff --cached --name-status -M \
    | awk -F '\t' -v p="$path" '$1 ~ /^R/ && $3 == p { print $2; exit }')
  [[ -n "$old" ]] || return 1
  [[ "$(git rev-parse "HEAD:$old")" == "$(git rev-parse ":$path")" ]]
}
head_archive_move() { # origin/master→HEAD 同内容归档改名不是新票号
  local base=$1 path=$2
  [[ "$path" == docs/tasks/done/T-* ]] || return 1
  local old
  old=$(git diff --name-status -M "$base" HEAD \
    | awk -F '\t' -v p="$path" '$1 ~ /^R/ && $3 == p { print $2; exit }')
  [[ -n "$old" ]] || return 1
  [[ "$(git rev-parse "$base:$old")" == "$(git rev-parse "HEAD:$path")" ]]
}

if [ "${1:-}" = "--staged" ]; then
  # staged 新路径：A/C 单路径、R 双路径取新（$NF）；M/D 不在滤内——存量票改注不触发本闸
  new_paths=$(git -c core.quotepath=off diff --cached --name-status --diff-filter=ACR \
    | awk -F'\t' '{print $NF}' | grep -E "$TICKET_RE" || true)
  if [ -z "$new_paths" ]; then
    exit 0
  fi

  status=0
  while IFS= read -r path; do
    key=$(printf '%s' "$path" | ticket_key)
    if staged_archive_move "$path"; then
      echo "NOTE(T-0163): $path 为原票归档改名，不视作新立票" >&2
      continue
    fi
    dups=$(num_files "$key")
    cnt=$(printf '%s\n' "$dups" | grep -c . || true)
    if [ "$cnt" -gt 1 ]; then
      echo "FAIL(T-0163): 票号 $key 撞号——index 内同号票 $cnt 张（双席取号竞态；后到让号重编，或挂起交决策裁）：" >&2
      printf '%s\n' "$dups" | sed 's/^/  /' >&2
      status=1
    fi
  done <<< "$new_paths"
  if [ "$status" -eq 1 ]; then
    exit 1
  fi

  # 取号推进提示（不拦）：本地最大号+1 非必对——双席并行远端为准（T-0163 票面）
  lmax=$(ticket_files docs/tasks | grep -E "$TICKET_RE" | ticket_num \
    | sort -n | tail -1 || true)
  echo "NOTE(T-0163): 新立票过本地唯一闸；本地最大票号=T-$lmax。取号非必对（双席视野差=撞号根因）——push 前 git fetch origin 并核 origin/master:docs/tasks/ 有无同号/更高号（远端为准）。" >&2
  exit 0
fi

if [ "${1:-}" = "--remote" ]; then
  # 远端对账模（T-0219——T-0217 撞实证：本地树陈时 --staged 只数 index 闸盲）。
  # 新立票面=staged ACR ∪ 推送区间（merge-base(origin/master,HEAD)..HEAD）ACR。
  # 判红三径：①远端同路径但内容异（同票分叉——先合流）②同号异名（真撞号）
  # ③新号 ≤远端最大号（本地视野陈，重取=scripts/next_ticket.sh）——但同径
  # 票面若在任一 origin 远端支已见=同票未并 master（T-0219 实证：决策支立票
  # 候并期号序前进），降级 NOTE 放行不误伤。
  if ! git fetch origin master --quiet 2>/dev/null; then
    echo "WARN(T-0219): git fetch 败——远端对账降级放行（离线推送不受阻）" >&2
    exit 0
  fi
  remote_files=$(git ls-tree -r --name-only origin/master -- docs/tasks/ 2>/dev/null || true)
  remote_keys=$(printf '%s\n' "$remote_files" | grep -E "$TICKET_RE" | ticket_key | sort -u || true)
  remote_nums=$(printf '%s\n' "$remote_files" | grep -E "$TICKET_RE" | ticket_num | sort -n | uniq || true)
  rmax=$(printf '%s\n' "$remote_nums" | tail -1)
  mb=$(git merge-base origin/master HEAD 2>/dev/null || true)
  new_paths=$( {
    git -c core.quotepath=off diff --cached --name-status --diff-filter=ACR 2>/dev/null \
      | awk -F'\t' '{print $NF}'
    [ -n "$mb" ] && git -c core.quotepath=off diff --name-status --diff-filter=ACR "$mb" HEAD 2>/dev/null \
      | awk -F'\t' '{print $NF}'
  } | grep -E "$TICKET_RE" | sort -u || true)
  if [ -z "$new_paths" ]; then
    echo "OK(T-0219): 无新立票——远端对账免（远端最大号 T-${rmax:-?}）" >&2
    exit 0
  fi
  status=0
  while IFS= read -r p; do
    [ -z "$p" ] && continue
    if staged_archive_move "$p" || head_archive_move "$mb" "$p"; then
      echo "NOTE(T-0219): $p 为 origin/master 已有票归档改名，票号身份未变" >&2
      continue
    fi
    key=$(printf '%s' "$p" | ticket_key)
    n=$(printf '%s' "$p" | ticket_num)
    if printf '%s\n' "$remote_files" | grep -qx "$p"; then
      if git diff --quiet origin/master HEAD -- "$p" 2>/dev/null; then
        echo "NOTE(T-0219): $p 远端同票同文——幂等放行" >&2
      else
        echo "FAIL(T-0219): $p 远端同号同径但内容异——同票分叉，先合流 origin/master 再推" >&2
        status=1
      fi
    elif printf '%s\n' "$remote_keys" | grep -Fxq "$key"; then
      echo "FAIL(T-0219): 票号 $key 远端已被他票占用（双席取号竞态——让号重编）:" >&2
      printf '%s\n' "$remote_files" | grep -E "/${key}(-|\.md$)" | sed 's/^/  /' >&2
      status=1
    elif [ -n "$rmax" ] && [ "$((10#$n))" -le "$((10#$rmax))" ]; then
      on_branch=0
      while IFS= read -r ref; do
        git cat-file -e "$ref:$p" 2>/dev/null && { on_branch=1; break; }
      done <<< "$(git for-each-ref 'refs/remotes/origin/t/*' --format='%(refname)')"
      if [ "$on_branch" = 1 ]; then
        echo "NOTE(T-0219): $p 远端支面已见（同票候并 master——T-0219 实证形）放行" >&2
      else
        echo "FAIL(T-0219): 新立票号 $key 的编号 $n ≤ 远端最大号 T-$rmax——本地视野陈，重取 scripts/next_ticket.sh" >&2
        status=1
      fi
    fi
  done <<< "$new_paths"
  if [ "$status" -eq 0 ]; then
    echo "OK(T-0219): 新立票号远端对账过（远端最大号 T-${rmax:-?}）" >&2
  fi
  exit "$status"
fi

# 无参：存量查重——exit 1=存在同号共存；存量处置归决策，本模式只出清单不拦入链
all_ticket_paths=$(ticket_files docs/tasks)
ticket_count=$(printf '%s\n' "$all_ticket_paths" | grep -c . || true)
echo "INFO: 票号扫描覆盖 $ticket_count 张（docs/tasks/ 与 done/）" >&2
dup_nums=$(printf '%s\n' "$all_ticket_paths" \
  | grep -E "$TICKET_RE" | ticket_key | sort | uniq -d || true)
status=0
while IFS= read -r key; do
  if [ -z "$key" ]; then
    continue
  fi
  cnt=$(num_files "$key" | grep -c . || true)
  echo "$key 同号共存 ${cnt} 张：" >&2
  num_files "$key" | sed 's/^/  /' >&2
  status=1
done <<< "$dup_nums"
if [ "$status" -eq 0 ]; then
  echo "OK: 票号全仓唯一" >&2
fi
exit "$status"
