#!/usr/bin/env bash
# deploy_oci.sh —— OCI 三臂（o-collecto/o-projector/o-research 裸二进制 systemd
# --user）部署唯一合法径（T-0400：手搓 GOARCH=arm64 build+scp+restart 无锁无
# 健康闸无出处闸无环约对账，collector 缺 SG_PG_DSN 崩环 76 轮断流 13min 实证）。
# 流（同 deploy_image.sh 纪律）：出处闸 → OCI 部署锁 → arm64 build（SHA 留痕）
# → staging scp+sha256 对拍 → 环约对账（码面 env 需求集 deploy/oci-env-contract.tsv
# vs 单元 Environment/EnvironmentFiles 键集——req=1 缺键预检拒换，值面补齐归
# 凭据域 escrow 径本脚本不碰 env 件）→ 备份 → 换件 → restart → 健康闸（单元
# active∧不翻饼；collector 断流面=PG-md md_ticks max(ts) 推进；projector=同断言
# 走隧道 SG_PG_DSN 侧）→ 败即回滚旧件。
# 用法：scripts/deploy_oci.sh [collecto|projector|research ...]（缺省三臂全换）
#   DRY_RUN=1 干跑（锁/船运/换件不执行，环约对账仍实读远端——只读面）
#   O_DEPLOY_ALLOW_UNPUSHED=1 放行未并 HEAD（回滚/热补径）；O_DEPLOY_FORCE=1 抢陈锁
set -euo pipefail
OCI="${O_OCI_HOST:-ubuntu@149.118.58.98}"
SSH="${O_OCI_SSH:-ssh}"
SCP="${O_OCI_SCP:-scp}"
SSHOPT="-o BatchMode=yes -o ConnectTimeout=10"
LOCK_DIR='.oci-deploy-lock'
CONTRACT="$(dirname "$0")/../deploy/oci-env-contract.tsv"
DRY="${DRY_RUN:-0}"
cd "$(dirname "$0")/.."
die() { echo "❌ [deploy_oci] $1" >&2; exit 1; }

declare -A UNIT=( [collecto]=o-collecto [projector]=o-projector [research]=o-research )
declare -A CMD=(  [collecto]=./cmd/collector [projector]=./cmd/projector [research]=./cmd/research )
# probe 类：md_ticks=盒本机 PG-md docker psql；sg_ticks=env 件 SG_PG_DSN 经容器
# psql 客端打隧道侧同断言；none=仅单元活性闸。
declare -A PROBE=( [o-collecto]=md_ticks [o-projector]=sg_ticks [o-research]=none )
ARMS=("$@"); [ ${#ARMS[@]} -gt 0 ] || ARMS=(collecto projector research)
SWAP_HAPPENED=0; SWAPPED=()
for a in "${ARMS[@]}"; do [ -n "${UNIT[$a]:-}" ] || die "未知臂 $a（合法: collecto|projector|research）"; done

# ── ① 出处闸（同 deploy_image.sh）─────────────────────────────
[ -z "$(git status --porcelain)" ] || die "工作树脏——commit/stash 后再部署"
git fetch origin master --quiet 2>/dev/null || echo "⚠️  [deploy_oci] git fetch 败——按本地 origin/master 快照判"
if ! git merge-base --is-ancestor HEAD origin/master; then
  [ "${O_DEPLOY_ALLOW_UNPUSHED:-0}" = 1 ] || die "HEAD 不在 origin/master——未并码禁部署（回滚/热补放行 O_DEPLOY_ALLOW_UNPUSHED=1）"
  echo "⚠️  [deploy_oci] O_DEPLOY_ALLOW_UNPUSHED=1——放行未并 HEAD"
fi
SHA=$(git rev-parse --short=12 HEAD); FULLSHA=$(git rev-parse HEAD)
SEAT="${O_DEPLOY_SEAT:-$(whoami)}@$(hostname -s 2>/dev/null || hostname)"
TOKEN="$SEAT-$$-$(date +%s)"; TS="$(date +%Y%m%d-%H%M%S)"

# ── ①b 谱系自检（T-0425 件③——两半部署制幂等径）：臂级构建源
# diff 证空=跳臂零烧。判式：diff(merge-base(臂部署SHA,HEAD)..HEAD)
# 碰本臂谱系行或不可判面→该臂照换；全件证外→跳过。谱系正本=
# deploy/deploy-faces.tsv（nondeploy/image_only/<arm> 三词表同源
# ——件缺=全面不可判全臂照换，滥换方向不省错方向）。臂部署 SHA=
# 盒上 $ep.sha 留痕件——不可读/谱系岔/diff 败=不可证空照换。
# O_DEPLOY_FORCE=1 压过自检（强制件不省——修件/热补径）。──
FACES_TSV="$(dirname "$0")/../deploy/deploy-faces.tsv"
faces_rows() { grep -v '^#' "$FACES_TSV" 2>/dev/null || true; } # tolerate: 件缺=否列空→全面不可判全臂照换（滥换方向不省错方向）
in_class() { # in_class <path> <class> → rc=0 路径属该类行
	local p=$1 want=$2 c g
	while IFS=$'\t' read -r c g; do
		{ [ "$c" = "$want" ] && [ -n "$g" ]; } || continue
		case "$p" in $g) return 0;; esac
	done < <(faces_rows)
	return 1
}
in_any_arm() { # in_any_arm <path> → rc=0 属任一 OCI 谱系行
	local p=$1 c g
	while IFS=$'\t' read -r c g; do
		case "$c" in collecto|projector|research|eventbus|sweeper) ;; *) continue;; esac
		case "$p" in $g) return 0;; esac
	done < <(faces_rows)
	return 1
}
arm_provably_clean() { # arm_provably_clean <arm> <path> → rc=0 证该件不改本臂
	local a=$1 p=$2
	in_class "$p" nondeploy && return 0
	in_class "$p" image_only && return 0
	in_any_arm "$p" && ! in_class "$p" "$a" && return 0
	return 1
}
arm_deployed_sha() { # arm_deployed_sha <unit> → stdout=臂部署 SHA（不可读=空）
	$SSH $SSHOPT "$OCI" "u=$1
f=\$(systemctl --user show \"\$u\" -p FragmentPath --value 2>/dev/null)
[ -n \"\$f\" ] || exit 0
e=\$(grep -m1 '^ExecStart=' \"\$f\" 2>/dev/null | sed 's/^ExecStart=//' | cut -d' ' -f1)
[ -n \"\$e\" ] || exit 0
e=\${e//%h/\$HOME}; e=\${e//%u/\$(id -un)}; e=\${e//%s/\${u%.service}}; e=\${e//%%/%}
sed -n 's/^sha=//p' \"\$e.sha\" 2>/dev/null | head -1" 2>/dev/null | head -1 || true # tolerate: ssh断/件缺=空→不可证空该臂照换（跳件只发证空面）
}
arm_stale() { # arm_stale <arm> <base> → rc=0 有件须换（diff 败=不可证空保守换）
	local a=$1 base=$2 p files
	files=$(git diff --name-only "$base..HEAD" 2>/dev/null) || return 0
	while IFS= read -r p; do
		[ -n "$p" ] || continue
		p=${p##* -> } # rename 形取新名
		arm_provably_clean "$a" "$p" || return 0
	done <<< "$files"
	return 1
}
if [ "${O_DEPLOY_FORCE:-0}" != 1 ]; then
	KEPT=()
	for a in "${ARMS[@]}"; do u="${UNIT[$a]}"
		s=$(arm_deployed_sha "$u")
		base=""
		[ -n "$s" ] && base=$(git merge-base "$s" HEAD 2>/dev/null || true) # tolerate: mb 败=谱系岔不可证空→该臂照换非静默省
		if [ -n "$base" ] && ! arm_stale "$a" "$base"; then
			echo "[deploy_oci] $u 谱系 diff 证空（部署 ${s:0:12} 起本臂构建源零动）——跳臂"
		else
			KEPT+=("$a")
		fi
	done
	ARMS=("${KEPT[@]}")
	[ ${#ARMS[@]} -gt 0 ] || { echo "[deploy_oci] 全臂谱系 diff 证空——幂等快退零烧"; exit 0; }
	echo "[deploy_oci] 谱系自检后待换臂: ${ARMS[*]}"
fi

# ── ② OCI 部署锁（盒上 mkdir 原子互斥——三臂同锁防双臂并发换件）──
LOCK_HELD=0
release_lock() { [ "$LOCK_HELD" = 1 ] || return 0
  $SSH $SSHOPT "$OCI" "grep -q '^token=$TOKEN\$' $LOCK_DIR/holder 2>/dev/null && rm -rf $LOCK_DIR" >/dev/null 2>&1 || true; } # tolerate: trap 释锁末环——锁目已失/网断=他席陈锁兜底接管，非静默病灶
trap 'release_lock' EXIT
if [ "$DRY" != 1 ]; then
  if $SSH $SSHOPT "$OCI" "mkdir $LOCK_DIR" 2>/dev/null; then LOCK_HELD=1
  else
    holder=$($SSH $SSHOPT "$OCI" "cat $LOCK_DIR/holder 2>/dev/null" || true) # tolerate: holder 件缺席=锁目残壳——显示 <空> 照走陈锁判
    age=$($SSH $SSHOPT "$OCI" "echo \$(( \$(date +%s) - \$(stat -c %Y $LOCK_DIR 2>/dev/null || echo 0) ))")
    if [ "${O_DEPLOY_FORCE:-0}" = 1 ] || [ "${age:-0}" -gt 1800 ]; then
      echo "⚠️  [deploy_oci] WARN 抢部署锁（age=${age}s）holder: ${holder:-<空>}" >&2
      $SSH $SSHOPT "$OCI" "rm -rf $LOCK_DIR && mkdir $LOCK_DIR" || die "抢锁败——他席恰持锁"
      LOCK_HELD=1
    else die "部署锁占用（age=${age}s/1800s）holder: ${holder:-<空>}——待释或 O_DEPLOY_FORCE=1"; fi
  fi
  $SSH $SSHOPT "$OCI" "printf 'seat=%s\nsha=%s\nts=%s\ntoken=%s\n' '$SEAT' '$SHA' \"\$(date -Is)\" '$TOKEN' > $LOCK_DIR/holder"
  echo "[deploy_oci] 锁持 $LOCK_DIR holder=$SEAT sha=$SHA"
fi

# ── ③ arm64 build + SHA 留痕件 → staging scp + sha256 对拍 ─────
# 完工回执（两半部署制 OCI 面同径——deploy_image deploy_event 同式）：
# feishu 直推+relay 留痕+ops_event 部署行。note 前缀 deploy-oci 与镜像
# 面 'deploy %' 行式互斥——窗面双源各取各不串味；SWAP_HAPPENED 闸
# 防谱系快退/纯预检退误落部署行。
oci_deploy_event() {
	local rc=$1
	[ "${SWAP_HAPPENED:-0}" = 1 ] || return 0
	local arms="${SWAPPED[*]:-}"
	echo "[deploy_oci] 完工回执 deploy-oci $SHA rc=$rc arms=$arms" >>"$HOME/devin-relay.log" 2>/dev/null || true # tolerate: relay 留痕行 best-effort
	timeout 15 /opt/meta/scripts/feishu-notify.sh \
		"[部署回执] deploy-oci $SHA rc=$rc arms=$arms" >>"$HOME/devin-relay.log" 2>&1 \
		|| echo "[deploy_oci] 完工投件败（回执缺，不影响部署 rc=$rc）" >&2
	local dsn
	dsn=$(grep '^FIN_PG_DSN=' /opt/o-research/env 2>/dev/null | cut -d= -f2- | tr -d '"')
	if [ -n "$dsn" ]; then
		timeout 15 psql "$dsn" -Atc \
			"INSERT INTO ops_event(seat,event,ticket,note) VALUES('DEPLOY','部署','','deploy-oci $SHA rc=$rc arms=$arms')" \
			>/dev/null 2>&1 \
			|| echo "[deploy_oci] ops_event 部署事件落簿败（不影响部署 rc=$rc）" >&2
	fi
	return 0
}
STAGE_DIR="/tmp/o-oci-stage.$$"; mkdir -p "$STAGE_DIR"
trap 'rc=$?; oci_deploy_event "$rc"; rm -rf "$STAGE_DIR"; release_lock' EXIT
declare -A STAGED
for a in "${ARMS[@]}"; do
  bin="$STAGE_DIR/${UNIT[$a]}"
  echo "[deploy_oci] 本地 arm64 build ${CMD[$a]} → ${UNIT[$a]} (sha=$SHA)"
  CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -trimpath -o "$bin" "${CMD[$a]}"
  printf 'sha=%s\nseat=%s\nts=%s\n' "$FULLSHA" "$SEAT" "$(date -Is)" > "$bin.sha"
  STAGED[$a]="$bin"
done
if [ "$DRY" != 1 ]; then
  for a in "${ARMS[@]}"; do
    b="${STAGED[$a]}"
    $SSH $SSHOPT "$OCI" "mkdir -p ~/o-staging"
    $SCP $SSHOPT -q "$b" "$b.sha" "$OCI:~/o-staging/"
    remote=$($SSH $SSHOPT "$OCI" "sha256sum ~/o-staging/$(basename "$b") | awk '{print \$1}'")
    [ "$remote" = "$(sha256sum "$b" | awk '{print $1}')" ] || die "$a staging sha256 不符——船运面不干净"
  done
fi

# ExecStart systemd specifier 远端语境展（T-0418——%h→该 ssh 户 $HOME/
# %u→属主名/%s→unit 名（去 .service 尾）；%%=字面 %。未识别 specifier/
# 包装形（env/bash -c/timeout 等）/多命令（; & |）=保守 die 报原值不猜——
# 与脚本 fail-closed 风同调）。RHOME/RUSER 懒取缓存。
RHOME=""; RUSER=""
expand_exec() { # expand_exec <raw ExecStart 值> <unit> → stdout=解析后绝对路径
  local raw=$1 unit=$2 tok
  if [[ "$raw" == *%* ]]; then
    [ -n "$RHOME" ] || RHOME=$($SSH $SSHOPT "$OCI" 'echo $HOME')
    [ -n "$RUSER" ] || RUSER=$($SSH $SSHOPT "$OCI" 'id -un')
    raw=${raw//%%/$'\x01'}; raw=${raw//%h/$RHOME}
    raw=${raw//%u/$RUSER}; raw=${raw//%s/${unit%.service}}
    if [[ "$raw" =~ (%[%a-zA-Z]) ]]; then
      die "$unit ExecStart 未识别 specifier ${BASH_REMATCH[1]}——保守拒猜（原值: $1）"
    fi
    raw=${raw//$'\x01'/%}
  fi
  [[ "$raw" == *[';&|']* ]] && die "$unit ExecStart 多命令/管道形——保守拒猜（原值: $1）"
  read -r tok _ <<<"$raw"
  while [[ "$tok" == [-+!@:]* ]]; do tok=${tok:1}; done # systemd 前缀修饰（-/+/!/@/:）剥
  [[ "$tok" == /* ]] || die "$unit ExecStart 解析非绝对路径——保守拒猜（原值: $1）"
  case "/$(basename "$tok")" in
    */env|*/bash|*/sh|*/dash|*/zsh|*/ksh|*/nice|*/ionice|*/timeout|*/setsid|*/chrt|*/systemd-cat|*/exec)
      die "$unit ExecStart 包装形 $(basename "$tok")——保守拒猜（原值: $1）";;
  esac
  printf '%s\n' "$tok"
}

# ── ④ 环约对账（远端实读——unit Environment/EnvironmentFiles 键集）──
# dry-run 亦执行（只读面=预检真值）。缺 req=1 键→拒换指路凭据域。
ENV_OK=1
declare -A EXECPATH
for a in "${ARMS[@]}"; do u="${UNIT[$a]}"
  frag=$($SSH $SSHOPT "$OCI" "systemctl --user show $u -p FragmentPath --value 2>/dev/null" || true) # tolerate: 读败=空值落下行 die 显报——失败面非静默吞
  [ -n "$frag" ] || die "$u 单元不在（盒面查 systemctl --user list-unit-files）"
  raw_exec=$($SSH $SSHOPT "$OCI" "grep -m1 '^ExecStart=' '$frag' | sed 's/^ExecStart=//'" || true) # tolerate: grep 空=ExecStart 行缺——下行 die 显报，失败面非静默吞
  [ -n "$raw_exec" ] || die "$u ExecStart 解析空（$frag 手核）"
  EXECPATH[$a]=$(expand_exec "$raw_exec" "$u")
  envkeys=$($SSH $SSHOPT "$OCI" "{ systemctl --user show $u -p Environment --value | tr ' ' '\n'; systemctl --user show $u -p EnvironmentFiles --value | tr ' ' '\n' | while read -r f; do [ -n \"\$f\" ] && sed -n 's/^[[:space:]]*\([A-Za-z_][A-Za-z_0-9]*\)=.*/\1/p' \"\$f\" 2>/dev/null; done; } | sed 's/=.*//' | grep -E '^[A-Za-z_]' | sort -u" || true) # tolerate: 键集读败=空集→全键判缺 fail-closed 拒换方向保守
  miss_req="" miss_opt=""
  while IFS=$'\t' read -r tu key req; do
    [[ "$tu" =~ ^#|^$ || "$tu" != "$u" ]] && continue
    if grep -qx "$key" <<<"$envkeys"; then continue; fi
    [ "$req" = 1 ] && miss_req="$miss_req $key" || miss_opt="$miss_opt $key"
  done < "$CONTRACT"
  [ -n "$miss_opt" ] && echo "⚠️  [deploy_oci] $u 可选键缺:${miss_opt}（降级面——WARN 不拒）"
  if [ -n "$miss_req" ]; then ENV_OK=0
    echo "❌ [deploy_oci] $u 环约对账红——必需 env 键缺:${miss_req}" >&2
    echo "    值面补齐=凭据域 escrow 径（业主/管理）——本脚本不碰 env 件（三禁）" >&2
  fi
done
[ "$ENV_OK" = 1 ] || die "环约对账不过——拒换件"
echo "[deploy_oci] 环约对账绿——需求集 ⊆ 单元 env 集"
[ "$DRY" = 1 ] && { echo "[deploy_oci] DRY_RUN=1 干跑止于此（锁/船运/换件未执行）"; exit 0; }

# ── ⑤ 备份 → 换件 → restart → 健康闸 → 败回滚 ─────────────────
tick_probe() { # tick_probe <arm> —— 写流断言 max(ts) 推进；0=推进 1=未推进
  local a=$1 u=${UNIT[$1]} q t0 t1
  case "${PROBE[$u]}" in
    md_ticks) q="docker exec o-postgres-md psql -U o -d o -tAc \"select coalesce(max(ts),'epoch'::timestamptz) from md_ticks\"";;
    sg_ticks) q="psql \"\$(grep '^SG_PG_DSN=' ~/o-projector.env|cut -d= -f2-|tr -d '\\\"')\" -tAc \"select coalesce(max(ts),'epoch'::timestamptz) from md_ticks\"";;
    *) return 0;;
  esac
  t0=$($SSH $SSHOPT "$OCI" "$q" 2>/dev/null || echo 0)
  for _ in $(seq 1 36); do sleep 5
    t1=$($SSH $SSHOPT "$OCI" "$q" 2>/dev/null || echo 0)
    [[ "$t1" > "$t0" ]] && { echo "[deploy_oci] $u 写流恢复 ${PROBE[$u]} max(ts): $t0→$t1"; return 0; }
  done
  return 1
}
rollback() { local a=$1 u=${UNIT[$a]} ep=${EXECPATH[$a]}
  echo "↩️  [deploy_oci] $u 回滚旧件" >&2
  $SSH $SSHOPT "$OCI" "install -m 755 '$ep.bak.$TS' '$ep' && systemctl --user restart $u"
  sleep 10
  st=$($SSH $SSHOPT "$OCI" "systemctl --user show $u -p ActiveState --value")
  echo "[deploy_oci] $u 回滚后 ActiveState=$st"
}
for a in "${ARMS[@]}"; do u="${UNIT[$a]}" ep="${EXECPATH[$a]}"
  $SSH $SSHOPT "$OCI" "cp -a '$ep' '$ep.bak.$TS'" # 备份
  $SSH $SSHOPT "$OCI" "install -m 755 ~/o-staging/${u} '$ep' && printf 'sha=%s\nseat=%s\nts=%s\n' '$FULLSHA' '$SEAT' '\$(date -Is)' > '$ep.sha'"
  $SSH $SSHOPT "$OCI" "systemctl --user restart $u"
  nr0=$($SSH $SSHOPT "$OCI" "systemctl --user show $u -p NRestarts --value" || echo 0); sleep 15
  st=$($SSH $SSHOPT "$OCI" "systemctl --user show $u -p ActiveState --value")
  nr1=$($SSH $SSHOPT "$OCI" "systemctl --user show $u -p NRestarts --value" || echo 0)
  if [ "$st" != active ] || [ "${nr1:-0}" -gt "${nr0:-0}" ]; then
    echo "❌ [deploy_oci] $u 活性闸红（ActiveState=$st NRestarts $nr0→$nr1）" >&2
    rollback "$a"; die "$u 换件败已回滚——journalctl --user -u $u 查因"
  fi
  if ! tick_probe "$a"; then
    echo "❌ [deploy_oci] $u 健康闸红——${PROBE[$u]} max(ts) 90s 未推进（断流面复现）" >&2
    rollback "$a"; die "$u 写流闸败已回滚"
  fi
  SWAP_HAPPENED=1; SWAPPED+=("$a")
  echo "[deploy_oci] $u 换件绿——ActiveState=$st 旧件备份 $ep.bak.$TS"
done
echo "[deploy_oci] done→$OCI（臂: ${ARMS[*]} sha=$SHA）"
