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
# psql 桩：按 -c SQL 词面应答；查询全落 PSQL_LOG 供断言。
# INSERT 走词法级校验（工单#29766）：剥合法串字面量后剩余裸词
# 只许 INSERT 结构词表——note 内嵌未转义 ' 会截断字面量产裸词
# （conclusion='failure'→failure 裸词）=真 psql 同款语法错→rc1
# 不落 PSQL_OK；''转义对合法不产裸词→落 PSQL_OK=「行落账」断言面。
# 引号计数偶/奇判不可信（嵌入引号恒成对——病灶恰在位置不在数量）。
# T_HB_FAIL=1 心跳写败径不变。-qAt 标志区分 esc() 主径（无）与
# escalate.sh 径（有）——T_ESC_PRI_FAIL 注败只咬主径,FB_FAIL_SCOPE 咬指定
# scope 的 escalate 径 INSERT;败件 rc1 不落 PSQL_OK=「行未落账」。
# 升件 dedup/pending 改有态——按 PSQL_OK 已落行应答（esc 兜底/自审案要
# 「写成败有后果」桩——静态 echo 0 验不出径差）。
sql=""; prev=""; qat=0
for a in "$@"; do [ "$a" = "-qAt" ] && qat=1; [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
pok="${PSQL_OK:-/dev/null}"
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
  INSERT*)
    if printf '%s' "$sql" | grep -q '心跳' && [ "${T_HB_FAIL:-0}" = 1 ]; then exit 1; fi
    if [ "${T_ESC_PRI_FAIL:-0}" = 1 ] && [ "$qat" = 0 ] \
      && printf '%s' "$sql" | grep -q 'esc_class,esc_scope'; then exit 1; fi
    if [ -n "${T_ESC_FB_FAIL_SCOPE:-}" ] && [ "$qat" = 1 ] \
      && printf '%s' "$sql" | grep -q ",'$T_ESC_FB_FAIL_SCOPE')"; then exit 1; fi
    stripped=$(printf '%s' "$sql" | sed -E "s/'([^']|'')*'//g")
    badbw=$(printf '%s\n' "$stripped" | tr -c 'A-Za-z_' '\n' \
      | grep -E '^[A-Za-z_]+$' | sort -u \
      | grep -vE '^(INSERT|INTO|ops_event|seat|event|ticket|note|esc_class|esc_scope|VALUES)$')
    [ -z "$badbw" ] || exit 1
    printf '%s\n' "$sql" >> "$pok"
    exit 0;;
  *deploy.ocore_window_utc*)  printf '%s\n' "${T_WIN:-19:00-21:00}";;
  *deploy.ocore_window_max_hold_h*) printf '%s\n' "${T_MAXH:-26}";;
  *o-core-clean:*) printf '%s\n' "${T_CLEANWM:-}";;
  *deploy-oci*) : ;;
  *ocore-window\ 入窗*) printf '%s\n' "${T_QUEUED:-0}";;
  *deploy-gate*) printf '%s\n' "${T_DIRTYN:-0}";;
  *regexp_match*o-core:*) printf '%s\n' "${T_DEPLOYED:-}";;
  *SELECT\ id*o-core:*) : ;;
  *event=\'升件\'*note\ LIKE*)
    pre=$(printf '%s' "$sql" | sed -n "s/.*note LIKE '\([^']*\)%.*/\1/p")
    { [ -n "$pre" ] && grep 'VALUES' "$pok" 2>/dev/null | grep -qF "$pre"; } \
      && echo 1 || echo 0;;
  *event=\'升件\'*esc_scope=*)
    scp=$(printf '%s' "$sql" | sed -n "s/.*esc_scope='\([^']*\)'.*/\1/p")
    { grep 'VALUES' "$pok" 2>/dev/null || true; } | grep -c ",'$scp')" || true;; # tolerate: pok 首拍未建/零 VALUES 行=正常空面,吞 grep rc1 取零计数
  *event=\'升件\'*) echo 0;;
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
cat > "$T/bin/gh" <<'EOF'
#!/usr/bin/env bash
# gh 桩：build-test check-runs 应答面——deploy_watch 取 --jq 后形
# 「<status> <conclusion>」（缺省绿；T_CIV 注入红/待闸态重放）。
case "$*" in
  *check-runs*) printf '%s\n' "${T_CIV:-completed success}";;
esac
exit 0
EOF
chmod +x "$T/bin/gh"

# ── 夹具：bare origin + master 对齐 clone（真 git；脚本/stub 入 base commit）──
new_repo() {
	rm -rf "$T/o" "$T/origin.git" "$T/psql.log" "$T/psql_ok.log" "$T/calls.log" "$T/out"
	git init --bare -q "$T/origin.git"
	git init -q -b master "$T/o"
	git -C "$T/o" config user.email t@t
	git -C "$T/o" config user.name t
	git -C "$T/o" remote add origin "$T/origin.git"
	mkdir -p "$T/o/docs" "$T/o/scripts/cognition"
	cp "$DW" "$T/o/scripts/deploy_watch.sh"
	cp "$HERE/cognition/escalate.sh" "$T/o/scripts/cognition/escalate.sh"
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
run_dw() { # $1=now epoch [$2=T_WIN] [$3=T_MAXH] [$4=T_CLEANWM] [$5=T_QUEUED] [$6=T_CIV]
	( cd "$T/o" && env PATH="$T/bin:$PATH" FIN_PG_DSN=stub REPORT_ENV_FILE="$T/envfile" \
		PSQL_LOG="$T/psql.log" PSQL_OK="$T/psql_ok.log" CALL_LOG="$T/calls.log" \
		O_DEPLOY_NOW_EPOCH="$1" T_DEPLOYED="$BASE" \
		T_WIN="${2:-19:00-21:00}" T_MAXH="${3:-26}" \
		T_CLEANWM="${4:-}" T_QUEUED="${5:-0}" T_DIRTYN="${T_DIRTYN:-0}" \
		T_CIV="${6:-completed success}" \
		T_ESC_PRI_FAIL="${T_ESC_PRI_FAIL:-0}" \
		T_ESC_FB_FAIL_SCOPE="${T_ESC_FB_FAIL_SCOPE:-}" \
		T_HB_LAST_STATE="${T_HB_LAST_STATE:-}" T_HB_LAST_AGE="${T_HB_LAST_AGE:-999999}" \
		bash scripts/deploy_watch.sh >"$T/out" 2>&1 )
}
now_of() { date -u -d "$1" +%s; }
has() { grep -q "$1" "$2" 2>/dev/null; }
# envfile——escalate.sh 兜底径只读 env 文件不读 env 变量,夹具共一件。
printf 'FIN_PG_DSN=stub\n' > "$T/envfile"

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

# ═══ ⑨ 脏树连拍升件（工单#25903）：deploy-gate 行 45min≥2→deploy_blocked ═══
echo "== ⑨ 脏树连拍升件 =="
new_repo; second_commit docs/note.md "doc" "2026-11-20T03:00:00"
T_DIRTYN=2 run_dw "$(now_of '2026-11-20 10:00:00')"
has 'deploy_blocked' "$T/psql.log" && has '脏树连拍拒部 2 次' "$T/psql.log"   && has 'stray=' "$T/psql.log" && ok "连拍2次→deploy_blocked 载 stray" || bad "连拍升件缺/载面不全"

# ═══ ⑩ 单拍不升（连拍阈防瞬时脏误报——单拍即升被否在案）═══
echo "== ⑩ 单拍不升 =="
new_repo; second_commit docs/note.md "doc" "2026-11-20T03:00:00"
T_DIRTYN=1 run_dw "$(now_of '2026-11-20 10:00:00')"
has '脏树连拍拒部' "$T/psql.log" && bad "单拍竟升件" || ok "单拍零升件"

# ═══ ⑪ CI红断升件引号面（工单#29766）：conclusion='failure' 嵌入
# 单引号曾断 SQL 字面量→INSERT 每拍写败被 tolerate 吞（deploy_ci_red
# 升件 0 行 vs CI红断心跳 1594 行/9h=信号面假净根因）。验收径=
# CIV=failure 重放→升件 INSERT 引号转义后落 PSQL_OK。═══
echo "== ⑪ CI红断引号面 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-20T03:00:00"
run_dw "$(now_of '2026-11-20 19:30:00')" 00:00-23:59 26 "" 0 "completed failure"
has 'deploy_ci_red' "$T/psql_ok.log" && ok "CI红断升件落账" || bad "CI红断升件缺/SQL破"
has "conclusion=''failure''" "$T/psql.log" && ok "引号已转义" || bad "引号未转义"
has 'CI红断' "$T/out" && ok "CI红断判词" || bad "CI红断判词缺"

# ═══ ⑫ esc 主径写败→escalate.sh 第二径兜底+心跳载 esc写败（工单#30086）═══
# 夹具 escalate.sh 用真件（scripts/cognition/ 拷入）——验的是真实第二径互
# 操作不是仿写;envfile 供 escalate 自取 DSN（它只读 env 文件不读 env 变量）。
echo "== ⑫ esc主径断兜底 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-20T03:00:00"
T_ESC_PRI_FAIL=1 run_dw "$(now_of '2026-11-20 19:30:00')" 00:00-23:59 26 "" 0 "completed failure"
has '部署/deploy_ci_red:——' "$T/psql_ok.log" && ok "兜底升件落账" || bad "兜底升件缺"
has '心跳 CI红断 esc写败' "$T/psql.log" && ok "心跳载 esc写败" || bad "心跳载败缺"
has 'esc缺' "$T/psql.log" && bad "兜底在案竟误补火" || ok "兜底在案零误补"

# ═══ ⑬ esc 全径断→态/实劈差自审 deploy_esc_missing 补火+心跳载 esc缺（工单#30087）═══
# FB_FAIL_SCOPE=deploy_ci_red：escalate 兜底径同 scope INSERT 败（词法合法
# 而写败——主径以外的第二断点注桩）→态/实劈差自审捕获补火。
echo "== ⑬ esc全径断自审 =="
new_repo; second_commit cmd/core/main.go "package main" "2026-11-20T03:00:00"
T_ESC_PRI_FAIL=1 T_ESC_FB_FAIL_SCOPE=deploy_ci_red \
	run_dw "$(now_of '2026-11-20 19:30:00')" 00:00-23:59 26 "" 0 "completed failure"
has 'deploy_esc_missing' "$T/psql_ok.log" && ok "自审补火落账" || bad "自审补火缺"
has '心跳 CI红断 esc缺' "$T/psql.log" && ok "心跳载 esc缺" || bad "心跳 esc缺缺"
has 'esc写败' "$T/psql.log" && ok "心跳载 esc写败" || bad "esc写败缺"

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
