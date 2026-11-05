# commit_report_hook.sh —— T-0317 返修①③ 零行兜底机制级解：
# git post-commit 钩——收口 commit（subject 含 [角色:施工][T-NNNN] 收口）
# 自动 report.sh 收口（自律条机器化——T-0315/T-0316 漏调零行案根修；
# hooks 目录 worktree 共享装一次全域生效；无匹配/调用败零阻 commit）。
# T-0325：生效件=scripts/hooks/post-commit（core.hooksPath 仓内径）——本件为
# 同源参考底稿，逻辑须与生效件一致。
#!/usr/bin/env bash
set -u
SUBJ=$(git log -1 --format=%s 2>/dev/null || true)
TID=$(echo "$SUBJ" | grep -oE 'T-[0-9]{4}' | head -1)
[ -n "$TID" ] || exit 0
# T-0340 词面收紧：正向锚=施工收口件（[角色:施工]前缀+「收口」词）。
# 负向排除返修不可采——真收口件「返修①收口——」自体含返修子串，排之=漏报；
# 决策裁决/验收件常含「返修①收口合master」「验收收口」字样，角色前缀隔之。
# T-0370 件②：收口词面再收紧——动作词位精确形（收口——/收口：/收口+/收口
# 结尾），名词连缀不误中（收口支SHA/收口待审/收口钩/收口戳/收口自报——
# T-0367 回填件「收口支SHA」误中指错 SHA 实证）。漏报向容忍：真收口件
# 规范形=「收口——」，异形漏报由 report.sh 收口手动径兜（红线：宁漏报）。
echo "$SUBJ" | grep -qF '[角色:施工]' || exit 0
echo "$SUBJ" | grep -qE '收口——|收口[：:]|收口$|收口[+＋]' || exit 0
SHA=$(git log -1 --format=%h 2>/dev/null || true)
~/bin/report.sh 收口 "$TID" "auto@$SHA" 2>/dev/null || true
exit 0
