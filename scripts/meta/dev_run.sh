#!/usr/bin/env bash
# dev_run.sh —— 无头施工席瞬态 run 壳（T-0324 拉起式钦定——systemd-run
# o-worker-<tid> unit 载本壳；scratch cwd 零仓件，worktree 职卡领单步自建）。
set -u
TID=${1:?usage: dev_run.sh <T-编号> [提示文]}
PROMPT=${2:-"读 /opt/o/skills/施工席/SKILL.md 施工协议 领 $TID 施工"}
# 钦定二择档：swe-2-high 默认|swe-2-medium——禁他档（业主令）。
MODEL=${DEV_RUN_MODEL:-swe-2-high}
case "$MODEL" in swe-2-high|swe-2-medium) ;; *) MODEL=swe-2-high;; esac

export O_SEAT=施工席
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

hb() { # $1=state——按票分行心跳（META-Dev-<tid> 并行施工互不盖）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('META-Dev-$TID','施工席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true
}

mkdir -p /var/tmp/o-worker && cd /var/tmp/o-worker || exit 1
hb alive
/home/mluser/.local/bin/devin -p --model "$MODEL" \
  --respect-workspace-trust false --permission-mode dangerous -- "$PROMPT"
RC=$?
[ $RC -eq 0 ] && hb alive || hb stalled
exit 0
