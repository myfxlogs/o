# commit_report_hook.sh —— T-0317 返修①③ 零行兜底机制级解：
# git post-commit 钩——收口 commit（subject 含 [角色:施工][T-NNNN] 收口）
# 自动 report.sh 收口（自律条机器化——T-0315/T-0316 漏调零行案根修；
# hooks 目录 worktree 共享装一次全域生效；无匹配/调用败零阻 commit）。
#!/usr/bin/env bash
set -u
SUBJ=$(git log -1 --format=%s 2>/dev/null || true)
TID=$(echo "$SUBJ" | grep -oE 'T-[0-9]{4}' | head -1)
[ -n "$TID" ] || exit 0
echo "$SUBJ" | grep -q '收口' || exit 0
SHA=$(git log -1 --format=%h 2>/dev/null || true)
~/bin/report.sh 收口 "$TID" "auto@$SHA" 2>/dev/null || true
exit 0
