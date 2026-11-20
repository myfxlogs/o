#!/usr/bin/env bash
# deploy_oci_test.sh —— deploy_oci.sh 工单#26150 三件桩测：
#   ① fin 迁移水位闸：live<reg→让位本拍零升件（零锁零换件零记档）；
#      live≥reg→闸绿照换。
#   ② 回滚同还 .sha：换件败→旧二进制+旧 sha 标件同还（假标不留）。
#   ③ 败 sha+臂记档 ops_event（rc≠0 恒记档含 fail=<arm>）→下拍同 sha
#      该臂 force 重试绕谱系自检（谱系证空臂仅 force 臂照换）。
# 桩面：ssh 双宿主（OCI=远端家目录件/scp 落件/systemctl/docker/psql 全
# 桩）、go build 桩、psql 本地桩（FIN_PG_DSN 径由 O_RESEARCH_ENV 指夹具）。
# 真 git bare origin+clone——谱系自检/出处闸走真判据；deploy_oci 拷入
# 夹具仓跑（cd 语义=仓根）。用法：bash scripts/deploy_oci_test.sh
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
T=$(mktemp -d /tmp/do-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

# ── 桩件铺装 ──
mkdir -p "$T/bin" "$T/state" "$T/home" "$T/remote_sg/home"
cat > "$T/bin/go" <<'EOF'
#!/usr/bin/env bash
# go build 桩：-o 下一参=产物径，内容=fakebin-<产物名> 供 sha256 对拍。
out=""; prev=""
for a in "$@"; do [ "$prev" = "-o" ] && out="$a"; prev="$a"; done
[ -n "$out" ] && printf 'fakebin-%s\n' "$(basename "$out")" > "$out"
exit 0
EOF
chmod +x "$T/bin/go"
cat > "$T/bin/scp" <<'EOF'
#!/usr/bin/env bash
# scp 桩：末参 host:~/o-staging/ → 拷源件入 OCI 假 staging。
dest="${!#}"
case "$dest" in *@*:*) ;; *) exit 1;; esac
ddir="${T_REMOTE_OCI:?}/home/o-staging"; mkdir -p "$ddir"
for a in "$@"; do case "$a" in -*) ;; *@*:*) ;; *) [ -f "$a" ] && cp "$a" "$ddir/";; esac; done
exit 0
EOF
chmod +x "$T/bin/scp"
cat > "$T/bin/ssh" <<'EOF'
#!/usr/bin/env bash
# ssh 桩双宿主：host 含 fakesgs→SG 假远端，其余→OCI 假远端。
# 远端命令在对应 home 下 eval——systemctl/docker/psql 桩经 PATH+态件应答。
host=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) shift 2;;   # -o <opt> 两参形（本仓 ssh/scp 调用面全用此形）
    -o*) shift;;    # -o<opt> 连写形兜底
    -*) shift;;
    *) host=$1; shift; break;;
  esac
done
cmd="$*"
case "$host" in *fakesgs*) R="${T_REMOTE_SG:?}";; *) R="${T_REMOTE_OCI:?}";; esac
cd "$R/home" && env -i PATH="$T_BIN_STUB:/usr/bin:/bin" HOME="$R/home" \
	T_FIN_LIVE="${T_FIN_LIVE:-}" T_STATE_DIR="${T_STATE_DIR:-}" T_REMOTE_OCI="${T_REMOTE_OCI:-}" \
	T_FAIL_ACTIVE="${T_FAIL_ACTIVE:-}" T_NRESTART_BUMP="${T_NRESTART_BUMP:-}" \
	T_RESTART_RC="${T_RESTART_RC:-}" bash -c "$cmd"
EOF
chmod +x "$T/bin/ssh"
cat > "$T/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
# systemctl --user 桩：show -p <prop> --value 读 $T_STATE_DIR 态件；restart 落痕+可翻态。
SD="${T_STATE_DIR:?}"; unit=""
[ "$1" = "--user" ] && shift
cmd=$1; shift
case "$cmd" in
  show) unit=$1; prop=""
    while [ $# -gt 0 ]; do [ "$1" = "-p" ] && { prop=$2; shift 2; continue; }; shift; done
    case "$prop" in
      FragmentPath)     echo "$T_REMOTE_OCI/home/.config/systemd/user/$unit.service";;
      Environment)      echo "";;
      EnvironmentFiles) echo "$T_REMOTE_OCI/home/$(basename "${unit%.service}").env";;
      NRestarts)        cat "$SD/nrestarts_$unit" 2>/dev/null || echo 0;;
      ActiveState)      cat "$SD/active_$unit" 2>/dev/null || echo active;;
      *) echo "";;
    esac;;
  restart)
    unit=$1; touch "$SD/restarted_$unit"
    case " ${T_FAIL_ACTIVE:-} " in *" $unit "*) echo failed > "$SD/active_$unit";; esac
    [ "${T_NRESTART_BUMP:-0}" = 1 ] && echo $(( $(cat "$SD/nrestarts_$unit" 2>/dev/null || echo 0) + 1 )) > "$SD/nrestarts_$unit"
    exit "${T_RESTART_RC:-0}";;
  *) exit 0;;
esac
EOF
chmod +x "$T/bin/systemctl"
cat > "$T/bin/docker" <<'EOF'
#!/usr/bin/env bash
# docker 桩：exec o-postgres-md …md_ticks=OCI 写流计数；exec o-postgres …
# schema_migrations=SG fin 水位（deploy_oci fin 闸唯一消费面）。
for a in "$@"; do
  case "$a" in
    o-postgres-md) f="${T_STATE_DIR:?}/tick"; n=$(( $(cat "$f" 2>/dev/null || echo 0) + 1 )); echo "$n" > "$f"; printf '%020d\n' "$n"; exit 0;;
    o-postgres)    echo "${T_FIN_LIVE:-0}"; exit 0;;
  esac
done
exit 0
EOF
chmod +x "$T/bin/docker"
cat > "$T/bin/psql" <<'EOF'
#!/usr/bin/env bash
# psql 桩三面：①FORCE_ARMS 探测(regexp_matches+deploy-oci)→T_FAIL_ARMS 行集
# ②ops_event INSERT→PSQL_LOG 落痕（T_PSQL_FAIL=1 拒写）③远端 sg_ticks
# 计数（md_ticks 字面量）→同 tick 文件。余查询空回零。
sql=""; prev=""
for a in "$@"; do case "$prev" in -c|-[A-Za-z]*c) sql="$a";; esac; prev="$a"; done
for a in "$@"; do case "$a" in *md_ticks*) f="${T_STATE_DIR:-/tmp}/tick"; n=$(( $(cat "$f" 2>/dev/null || echo 0) + 1 )); echo "$n" > "$f"; printf '%020d\n' "$n"; exit 0;; esac; done
case "$sql" in
  *regexp_matches*deploy-oci*) printf '%s\n' "${T_FAIL_ARMS:-}" | sed '/^$/d'; exit 0;;
  INSERT*ops_event*) printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"; exit "${T_PSQL_FAIL:-0}";;
  *) exit 0;;
esac
EOF
chmod +x "$T/bin/psql"
export T_BIN_STUB="$T/bin" T_REMOTE_OCI="$T/remote_oci" T_REMOTE_SG="$T/remote_sg" \
	T_STATE_DIR="$T/state" PSQL_LOG="$T/psql.log"

# ── 夹具仓：deploy_oci 真件+依赖件入 git，bare origin 对齐 ──
new_repo() { # $1=二提交改动件路径（须落在三臂共有谱系行）
	rm -rf "$T/o" "$T/origin.git"; git init --bare -q "$T/origin.git"
	git init -q -b master "$T/o"; git -C "$T/o" config user.email t@t; git -C "$T/o" config user.name t
	git -C "$T/o" remote add origin "$T/origin.git"
	mkdir -p "$T/o/scripts" "$T/o/deploy" "$T/o/internal/store/pgstore" "$T/o/docs" "$T/o/$(dirname "$1")"
	cp "$HERE/deploy_oci.sh" "$HERE/arm_faces_lib.sh" "$HERE/oci_ssh_route.sh" "$T/o/scripts/"
	cp "$HERE/../deploy/deploy-faces.tsv" "$HERE/../deploy/oci-env-contract.tsv" "$T/o/deploy/"
	cp "$HERE/../internal/store/pgstore/migrate_table.go" "$T/o/internal/store/pgstore/"
	echo base > "$T/o/docs/base.md"
	git -C "$T/o" add -A; git -C "$T/o" commit -qm init; git -C "$T/o" push -qu origin master
	BASE_FULL=$(git -C "$T/o" rev-parse HEAD)
	echo change > "$T/o/$1"; git -C "$T/o" add -A; git -C "$T/o" commit -qm change; git -C "$T/o" push -qu origin master
	HEAD_FULL=$(git -C "$T/o" rev-parse HEAD); HEAD_SHA=$(git -C "$T/o" rev-parse --short=12 HEAD)
}

# ── OCI 假远端家目录：三臂 unit/env/二进制/.sha ──
setup_remote() { # $1=各臂部署 sha（全值同——调用方按案备）
	local dsha=$1 a u
	rm -rf "$T/remote_oci" "$T/state"; mkdir -p "$T/state" "$T/remote_sg/home"
	for a in collecto projector research; do
		u="o-$a"
		mkdir -p "$T/remote_oci/home/.config/systemd/user" "$T/remote_oci/home/o-staging" "$T/remote_oci/arms"
		printf '[Service]\nExecStart=%s\n' "$T/remote_oci/arms/$u" > "$T/remote_oci/home/.config/systemd/user/$u.service"
		printf 'OLD-%s\n' "$a" > "$T/remote_oci/arms/$u"
		printf 'sha=%s\n' "$dsha" > "$T/remote_oci/arms/$u.sha"
		case "$a" in
			collecto)  printf 'SG_PG_DSN=postgres://x\n' > "$T/remote_oci/home/o-collecto.env";;
			projector) printf 'MD_PG_DSN=postgres://x\nSG_PG_DSN=postgres://x\n' > "$T/remote_oci/home/o-projector.env";;
			research)  printf 'SG_PG_DSN=postgres://x\nFIN_PG_DSN=postgres://x\n' > "$T/remote_oci/home/o-research.env";;
		esac
		echo active > "$T/state/active_$u"; echo 0 > "$T/state/nrestarts_$u"
	done
}

run_deploy() { # $* = 臂参；env 面由调用方先 export；stdout=rc=N 行
	: > "$PSQL_LOG"
	PATH="$T/bin:$PATH" HOME="$T/home" O_OCI_HOST=ubuntu@fakeoci O_SG_HOST=root@fakesgs \
	O_OCI_JARG="-o ProxyJump=none" O_RESEARCH_ENV="$T/research.env" \
	FEISHU_MANAGE_ENV=/nonexistent \
	bash "$T/o/scripts/deploy_oci.sh" "$@" > "$T/out" 2>&1
	echo "rc=$?"
}
printf 'FIN_PG_DSN=postgres://stub\n' > "$T/research.env"

# ── T1 fin 水位闸拦截：live<reg→让位零升件 ──
echo "T1 fin 水位闸拦截（live<reg 让位本拍）"
new_repo internal/platform/config/dummy.go
setup_remote "$BASE_FULL"
REG_FIN=$(grep -oE '\{[0-9]+, "fin_[0-9]+' "$T/o/internal/store/pgstore/migrate_table.go" | grep -oE '[0-9]+' | sort -n | tail -1)
T_FIN_LIVE=1 run_deploy > "$T/rc1"
if grep -q 'fin 迁移水位闸未过' "$T/out" && ! grep -q 'done→' "$T/out" \
	&& [ "$(cat "$T/remote_oci/arms/o-collecto")" = "OLD-collecto" ] \
	&& [ ! -d "$T/remote_oci/home/.oci-deploy-lock" ] && [ ! -s "$PSQL_LOG" ]; then
	ok "T1 水位闸拦截——让位零锁零换件零记档"
else bad "T1 水位闸拦截"; cat "$T/out"; fi

# ── T2 水位闸放行+三臂换件绿+rc=0 记档 ──
echo "T2 水位闸放行+全臂换件绿"
new_repo internal/platform/config/dummy.go
setup_remote "$BASE_FULL"
T_FIN_LIVE=$((REG_FIN+5)) run_deploy > "$T/rc2"
if [ "$(cat "$T/rc2")" = "rc=0" ] && grep -q 'done→' "$T/out" \
	&& grep -q 'fin 迁移水位闸绿' "$T/out" \
	&& [ "$(cat "$T/remote_oci/arms/o-collecto")" = "fakebin-o-collecto" ] \
	&& [ "$(cat "$T/remote_oci/arms/o-research")" = "fakebin-o-research" ] \
	&& grep -q "deploy-oci $HEAD_SHA rc=0" "$PSQL_LOG" \
	&& grep -q "^sha=$HEAD_FULL" "$T/remote_oci/arms/o-collecto.sha" \
	&& [ ! -d "$T/remote_oci/home/.oci-deploy-lock" ]; then
	ok "T2 闸绿换件——三臂新件+.sha 新标+rc=0 记档+锁释"
else bad "T2 闸绿换件"; cat "$T/out"; fi

# ── T3 换件败回滚同还 .sha+fail= 臂记档 ──
echo "T3 换件败回滚——.sha 同还消假标+fail=research 记档"
new_repo internal/platform/config/dummy.go
setup_remote "$BASE_FULL"
T_FIN_LIVE=$((REG_FIN+5)) T_FAIL_ACTIVE=o-research run_deploy > "$T/rc3"
if [ "$(cat "$T/rc3")" = "rc=1" ] \
	&& [ "$(cat "$T/remote_oci/arms/o-research")" = "OLD-research" ] \
	&& grep -q "^sha=$BASE_FULL" "$T/remote_oci/arms/o-research.sha" \
	&& grep -q "deploy-oci $HEAD_SHA rc=1.*fail=research" "$PSQL_LOG" \
	&& [ "$(cat "$T/remote_oci/arms/o-collecto")" = "fakebin-o-collecto" ]; then
	ok "T3 回滚同还 .sha——旧件旧标复归+fail=research 记档+前臂已换不撤"
else bad "T3 回滚同还 .sha"; cat "$T/out"; tail -3 "$PSQL_LOG"; fi

# ── T4 败臂 force 重试：谱系证空臂仅 fail 臂照换 ──
echo "T4 败臂 force 重试自复（谱系证空+fail=research→research 照换余跳）"
new_repo internal/platform/config/dummy.go
setup_remote "$HEAD_FULL"   # 三臂部署 sha=HEAD→谱系 diff 证空全应跳
T_FIN_LIVE=$((REG_FIN+5)) T_FAIL_ARMS=research run_deploy > "$T/rc4"
if [ "$(cat "$T/rc4")" = "rc=0" ] \
	&& grep -q 'force 重试' "$T/out" \
	&& [ -f "$T/state/restarted_o-research" ] \
	&& [ ! -f "$T/state/restarted_o-collecto" ] && [ ! -f "$T/state/restarted_o-projector" ] \
	&& grep -q '谱系 diff 证空' "$T/out"; then
	ok "T4 force 重试——证空面仅 fail 臂照换"
else bad "T4 force 重试"; cat "$T/out"; fi

# ── T5 换件前段败（env 对账红）rc≠0 恒记档无 fail= ──
echo "T5 换件前段败恒记档（env 对账红→rc=1 无 fail= 不触发 force）"
new_repo internal/platform/config/dummy.go
setup_remote "$BASE_FULL"
sed -i '/FIN_PG_DSN/d' "$T/remote_oci/home/o-research.env"  # research req 键缺→对账红
T_FIN_LIVE=$((REG_FIN+5)) run_deploy > "$T/rc5"
if [ "$(cat "$T/rc5")" = "rc=1" ] \
	&& grep -q "deploy-oci $HEAD_SHA rc=1" "$PSQL_LOG" \
	&& ! grep -q "fail=" "$PSQL_LOG" \
	&& [ "$(cat "$T/remote_oci/arms/o-collecto")" = "OLD-collecto" ]; then
	ok "T5 前段败记档——rc=1 落账无 fail= 臂"
else bad "T5 前段败记档"; cat "$T/out"; tail -3 "$PSQL_LOG"; fi

echo "── deploy_oci_test: $PASS 过 $FAILN 败 ──"
[ "$FAILN" = 0 ]
