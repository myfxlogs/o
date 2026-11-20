#!/usr/bin/env bash
# deploy_watch_test.sh —— o-core 换件部署窗桩测（工单#25760 验收）：
# 桩 psql/ssh/go + 本地 bare origin+clone——真 deploy_watch.sh 拷入夹具跑。
# 七态：①窗外谱系证内→入窗队列不换 ②窗内→照换 ③谱系证空→水位跳部署
# 不回归 ④水位在案→跳部署 ⑤越窗未放→升件 ⑥跨午夜窗判据 ⑦窗配置非法
# →fail-closed 不换钱径+升件。桩 deploy_image 只在换件时落痕（CALL_LOG）。
# 用法：bash scripts/deploy_watch_test.sh
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
DW="$HERE/deploy_watch.sh"
T=$(mktemp -d /tmp/dw-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

# ── 桩件铺装 ──
mkdir -p "$T/bin"
cat > "$T/bin/psql" <<'EOF'
#!/usr/bin/env bash
# psql 桩：按 -c SQL 词面应答；查询全落 PSQL_LOG 供断言（INSERT 亦记）。
sql=""; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
  INSERT*心跳*'') [ "${T_HB_FAIL:-0}" = 1 ] && exit 1; exit 0;;
  INSERT*) exit 0;;
  *deploy.ocore_window_utc*)  printf '%s\n' "${T_WIN:-19:00-21:00}";;
  *deploy.ocore_window_max_hold_h*) printf '%s\n' "${T_MAXH:-26}";;
  *o-core-clean:*) printf '%s\n' "${T_CLEANWM:-}";;
  *deploy-oci*) : ;;
  *ocore-window\ 入窗*) printf '%s\n' "${T_QUEUED:-0}";;
  *regexp_match*o-core:*) printf '%s\n' "${T_DEPLOYED:-}";;
  *SELECT\ id*o-core:*) : ;;
  *event='升件'*) echo 0;;
  *split_part*note*event='心跳'*) printf '%s|%s\n' "${T_HB_LAST_STATE:-}" "${T_HB_LAST_AGE:-999999}";;
  *) : ;;
esac
exit 0
EOF
chmod +x "$T/bin/psql"
cat > "$T/bin/ssh" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
chmod +x "$T/bin/ssh"
cat > "$T/bin/go" <<'EOF'
#!/usr/bin/env bash
# go list -deps 桩：cmd/core 闭包只留主件+一包（diff 落 cmd/core/ 即证内）。
printf 'o/cmd/core\no/internal/platform/config\n'
EOF
chmod +x "$T/bin/go"

# ── 夹具：bare origin + master 对齐 clone（真 git；脚本/stub 入 base commit）──
new_repo() {
	rm -rf "$T/o" "$T/origin.git" "$T/psql.log" "$T/calls.log" "$T/out"
	git init --bare -q "$T/origin.git"
	git init -q -b master "$T/o"
	git -C "$T/o" config user.email t@t
	git -C "$T/o" config user.name t
	git -C "$T/o" remote add origin "$T/origin.git"
	mkdir -p "$T/o/docs" "$T/o/scripts"
	cp "$DW" "$T/o/scripts/deploy_watch.sh"
	cat > "$T/o/scripts/arm_faces_lib.sh" <<'EOF'
arm_stale() { return 1; }
EOF
	cat > "$T/o/scripts/deploy_image.sh" <<'EOF'
#!/usr/bin/env bash
echo called >> "${CALL_LOG:-/dev/null}"
exit 0
EOF
	echo base > "$T/o/docs/base.md"
	git -C "$T/o" add -A
	git -C "$T/o" commit -qm init
	git -C "$T/o" push -qu origin master
	BASE=$(git -C "$T/o" rev-parse --short=12 master)
}
second_commit() { # $1=path $2=content [$3=commit date ISO]
	mkdir -p "$T/o/$(dirname "$1")"
	echo "$2" > "$T/o/$1"
	git -C "$T/o" add -A
	if [ -n "${3:-}" ]; then
		GIT_AUTHOR_DATE="$3" GIT_COMMITTER_DATE="$3" git -C "$T/o" commit -qm second
	else
		git -C "$T/o" commit -qm second
	fi
	git -C "$T/o" push -qu origin master
	HEAD2=$(git -C "$T/o" rev-parse --short=12 master)
}
run_dw() { # $1=now epoch [$2=T_WIN] [$3=T_MAXH] [$4=T_CLEANWM] [$5=T_QUEUED]
	( cd "$T/o" && env PATH="$T/bin:$PATH" FIN_PG_DSN=stub REPORT_ENV_FILE=/dev/null \
		PSQL_LOG="$T/psql.log" CALL_LOG="$T/calls.log" \
		O_DEPLOY_NOW_EPOCH="$1" T_DEPLOYED="$BASE" \
		T_WIN="${2:-19:00-21:00}" T_MAXH="${3:-26}" \
		T_CLEANWM="${4:-}" T_QUEUED="${5:-0}" \
		T_HB_LAST_STATE="${T_HB_LAST_STATE:-}" T_HB_LAST_AGE="${T_HB_LAST_AGE:-999999}" \
		bash scripts/deploy_watch.sh >"$T/out" 2>&1 )
}
now_of() { date -u -d "$1" +%s; }
has() { grep -q "$1" "$2" 2>/dev/null; }

# ═══ ① 窗外谱系证内 → 入窗队列，不换 ═══
echo "== ① 窗外入窗队列 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-20T03:00:00"
run_dw "$(now_of '2026-11-20 10:00:00')"
has 'ocore-window 入窗' "$T/psql.log" && ok "入窗队列行在账" || bad "入窗队列行缺"
[ ! -s "$T/calls.log" ] && ok "窗外未调 deploy_image" || bad "窗外竟换件"
has '窗外（窗=19:00-21:00 UTC）' "$T/out" && ok "窗外判词" || bad "窗外判词缺"

# ═══ ② 窗内 → 照换，不排队 ═══
echo "== ② 窗内照换 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-20T03:00:00"
run_dw "$(now_of '2026-11-20 19:30:00')"
[ -s "$T/calls.log" ] && ok "窗内调 deploy_image" || bad "窗内未换件"
has 'ocore-window 入窗' "$T/psql.log" && bad "窗内竟入窗队列" || ok "窗内不入窗队列"
has '起部署' "$T/out" && ok "窗内起部署判词" || bad "窗内起部署判词缺"

# ═══ ③ 谱系证空 → 水位跳部署不回归 ═══
echo "== ③ 谱系证空水位 =="
new_repo; second_commit docs/note.md "doc" "2026-11-20T03:00:00"
run_dw "$(now_of '2026-11-20 10:00:00')"
has 'o-core-clean:' "$T/psql.log" && ok "o-core-clean 水位落" || bad "o-core-clean 水位缺"
[ ! -s "$T/calls.log" ] && ok "证空零换件" || bad "证空竟换件"
has 'ocore-window 入窗' "$T/psql.log" && bad "证空竟入窗队列" || ok "证空不入窗队列"

# ═══ ④ 水位在案 → 跳部署 ═══
echo "== ④ 水位在案跳部署 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-20T03:00:00"
run_dw "$(now_of '2026-11-20 10:00:00')" 19:00-21:00 26 "$HEAD2"
has '谱系证空水位在案' "$T/out" && ok "水位在案判词" || bad "水位在案判词缺"
[ ! -s "$T/calls.log" ] && ok "水位在案零换件" || bad "水位在案竟换件"

# ═══ ⑤ 越窗未放 → 升件 ═══
echo "== ⑤ 越窗未放升件 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-19T04:00:00"
run_dw "$(now_of '2026-11-20 10:00:00')"
has 'deploy_ocore_window_overdue' "$T/psql.log" && ok "越窗升件在账" || bad "越窗升件缺"
[ ! -s "$T/calls.log" ] && ok "越窗仍未换件" || bad "越窗竟换件"

# ═══ ⑥ 跨午夜窗内判据 ═══
echo "== ⑥ 跨午夜窗 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-20T03:00:00"
run_dw "$(now_of '2026-11-21 00:30:00')" 23:00-01:00
[ -s "$T/calls.log" ] && ok "跨午夜窗内照换" || bad "跨午夜窗内未换"

# ═══ ⑦ 窗配置非法 → fail-closed+升件 ═══
echo "== ⑦ 窗配置非法 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-20T03:00:00"
run_dw "$(now_of '2026-11-20 10:00:00')" bogus
has 'deploy_ocore_window_config' "$T/psql.log" && ok "配置非法升件在账" || bad "配置非法升件缺"
[ ! -s "$T/calls.log" ] && ok "配置非法不换钱径" || bad "配置非法竟换件"

# ═══ ⑧ 心跳面（工单#25803）：每拍写 ops_event seat=DEPLOY event=心跳 ═══
echo "== ⑧ 心跳面 =="
new_repo; second_commit docs/note.md "doc" "2026-11-20T03:00:00"
run_dw "$(now_of '2026-11-20 10:00:00')"
has "VALUES('DEPLOY','心跳'" "$T/psql.log" && ok "心跳行落账" || bad "心跳行缺"
has "心跳 证空跳部署" "$T/psql.log" && ok "判词随行（证空跳部署）" || bad "判词随行缺"
has '心跳: 证空跳部署' "$T/out" && ok "say 心跳判词" || bad "say 心跳缺"
# 写败容忍：psql 桩 INSERT 败→脚本仍 rc0（心跳非钱径不翻部署判）
new_repo; second_commit docs/note.md "doc" "2026-11-20T03:00:00"
( cd "$T/o" && env PATH="$T/bin:$PATH" FIN_PG_DSN=stub REPORT_ENV_FILE=/dev/null \
	PSQL_LOG="$T/psql.log" CALL_LOG="$T/calls.log" O_DEPLOY_NOW_EPOCH="$(now_of '2026-11-20 10:00:00')" \
	T_DEPLOYED="$BASE" T_WIN=19:00-21:00 T_MAXH=26 T_CLEANWM= T_QUEUED=0 \
	T_HB_FAIL=1 bash scripts/deploy_watch.sh >"$T/out" 2>&1 ); rc=$?
[ "$rc" = 0 ] && ok "心跳写败不翻部署判 rc=0" || bad "心跳写败翻判 rc=$rc"
has '心跳写败' "$T/out" && ok "写败如实报" || bad "写败静默"

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
