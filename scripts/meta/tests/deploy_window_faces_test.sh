#!/usr/bin/env bash
# deploy_window_faces_test.sh —— T-0425 两半部署制窗面桩测：
# ①红桩 diff 只碰 OCI 构建源→应发 faces=oci+半截升件行实落
# ②红桩 diff 只碰镜像专属面→faces=image OCI 面不误发
# ③双面批→faces=image,oci（零半截——同基线）
# ④双面俱空→批空 exit10 faces 空行
# ⑤OCI 基线三源皆败→面不可判 oci 保守计入（钦定按双发）
# 桩面：TMP git 仓 c0→c1(internal/projector)→c2(web) 线史；
# env 覆写跳双源；systemctl/ssh/psql PATH 桩+dm_queue/ops_esc ENV 桩
# ——不碰真盒真库真 systemd。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf -- "$TMP"' EXIT

# ── 桩仓：c0 base → c1 internal/projector → c2 web ───────────
REPO="$TMP/repo"; mkdir -p "$REPO/deploy" "$REPO/internal/projector" "$REPO/web"
cp "$ROOT/deploy/deploy-faces.tsv" "$REPO/deploy/"
git -C "$REPO" init -q -b master
G() { git -C "$REPO" -c user.name=t -c user.email=t@t "$@"; }
echo a > "$REPO/base.txt"; G add -A; G commit -qm c0; C0=$(G rev-parse HEAD)
echo x > "$REPO/internal/projector/tick.go"; G add -A; G commit -qm "c1 [T-0001]"; C1=$(G rev-parse HEAD)
echo x > "$REPO/web/w.ts"; G add -A; G commit -qm "c2 [T-0002]"; C2=$(G rev-parse HEAD)
# T-0437 分面桩支：t/0900=web 件（img 单面）/t/0901=internal 件（双面）/
# t/0903=web 件（待裁面用）；T-0902 无支无树=unborn（不可判→双挡）
G checkout -qb t/0900 master; echo y > "$REPO/web/w2.ts"; G add -A; G commit -qm "t0900 web"
G checkout -qb t/0901 master; echo z > "$REPO/internal/projector/extra.go"; G add -A; G commit -qm "t0901 internal"
G checkout -qb t/0903 master; echo w > "$REPO/web/w3.ts"; G add -A; G commit -qm "t0903 web"
G checkout -q master

# ── 桩面 ─────────────────────────────────────────────────
echo 'FIN_PG_DSN=postgres://127.0.0.1:1/x' > "$TMP/env"
DMQ_CTRL="$TMP/dmq_ctrl"; : > "$DMQ_CTRL" # 行1=fail(非零码) 行2=pending 票单
cat > "$TMP/dmq.sh" <<EOF
#!/usr/bin/env bash
[ -s "$DMQ_CTRL" ] || exit 0
if head -1 "$DMQ_CTRL" | grep -q '^fail'; then
  exit \$(head -1 "$DMQ_CTRL" | grep -oE '[0-9]+')
fi
cat "$DMQ_CTRL"
EOF
chmod +x "$TMP/dmq.sh"
UNIT_CTL="$TMP/units"; : > "$UNIT_CTL" # 在飞 o-worker 单位行（systemctl 桩吐）
mkdir -p "$TMP/bin"
cat > "$TMP/bin/systemctl" <<EOF
#!/usr/bin/env bash
[ -s "$UNIT_CTL" ] && cat "$UNIT_CTL"
exit 0
EOF
chmod +x "$TMP/bin/systemctl"
for b in ssh psql; do
	printf '#!/usr/bin/env bash\nexit 0\n' > "$TMP/bin/$b"; chmod +x "$TMP/bin/$b"
done
ESC_CALLS="$TMP/esc_calls"; : > "$ESC_CALLS"
cat > "$TMP/esc.sh" <<EOS
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$ESC_CALLS"
EOS
chmod +x "$TMP/esc.sh"

run() { # run <img_sha> <oci_sha|''> → stdout=壳输出；rc 经 ||真 不炸 set -e
	( env O_DEPLOY_WINDOW_REPO="$REPO" O_MASTER_REF=master \
		O_DEPLOYED_SHA="$1" O_OCI_DEPLOYED_SHA="$2" \
		DM_ENV_FILE="$TMP/env" O_DM_QUEUE_SH="$TMP/dmq.sh" \
		OPS_ESC_SH="$TMP/esc.sh" PATH="$TMP/bin:$PATH" \
		bash "$ROOT/scripts/meta/deploy_window_due.sh" )
}
state()  { sed -n 's/^state=//p'  <<<"$1" | head -1; }
faces()  { sed -n 's/^faces=//p'  <<<"$1" | head -1; }
esc_has(){ grep -qF "$1" "$ESC_CALLS"; }
fail() { echo "❌ $*" >&2; exit 1; }

# ①红桩——镜像基线=HEAD（镜像面无批）OCI 基线=c0（谱系陈）：
# diff 只碰 OCI 构建源→应发 faces=oci；oci≺img lag-range 碰 OCI→半截升件实落。
: > "$ESC_CALLS"; rc=0; out=$(run "$C2" "$C0") || rc=$?
[ "$rc" -eq 0 ] || fail "①rc=$rc≠0: $out"
[ "$(state "$out")" = 应发 ] || fail "①state≠应发: $out"
[ "$(faces "$out")" = oci ] || fail "①faces≠oci: $out"
esc_has 半截部署 || fail "①半截部署升件行未落"

# ②红桩——双基线同 c1：diff 只碰镜像专属（web）→faces=image OCI 面不误发；同基线零半截。
: > "$ESC_CALLS"; out=$(run "$C1" "$C1") || rc=$?
[ "$rc" -eq 0 ] || fail "②rc=$rc≠0: $out"
[ "$(faces "$out")" = image ] || fail "②faces≠image（OCI 面误发）: $out"
esc_has 半截部署 && fail "②同基线竟升半截部署"
! esc_has 半截部署 || fail "②半截误升"

# ③双面批——双基线同 c0：projector+web 并碰→faces=image,oci；等基线不升半截。
: > "$ESC_CALLS"; out=$(run "$C0" "$C0") || rc=$?
[ "$rc" -eq 0 ] || fail "③rc=$rc≠0: $out"
[ "$(faces "$out")" = image,oci ] || fail "③faces≠image,oci: $out"
! esc_has 半截部署 || fail "③等基线半截误升"

# ④双面俱空——双基线同 HEAD→批空 exit10 faces 空行。
: > "$ESC_CALLS"; rc=0; out=$(run "$C2" "$C2") || rc=$?
[ "$rc" -eq 10 ] || fail "④rc=$rc≠10: $out"
[ "$(state "$out")" = 批空 ] || fail "④state≠批空: $out"
[ -z "$(faces "$out")" ] || fail "④faces 非空: $out"

# ⑤OCI 基线不可判——env 空+ssh/psql 桩全空→oci 保守计入 faces（按双发）。
# 顺桩：纯不可判批 EARLIEST=0→兜底节拍无锚不压过（DEPLOY-WINDOW-OVERRUN 误升防案）。
: > "$ESC_CALLS"; rc=0; out=$(run "$C2" "") || rc=$?
[ "$rc" -eq 0 ] || fail "⑤rc=$rc≠0: $out"
[ "$(faces "$out")" = oci ] || fail "⑤faces≠oci（不可判未保守计）: $out"
! esc_has 半截部署 || fail "⑤不可判竟升半截（无谱系可证不升）"
! esc_has DEPLOY-WINDOW-OVERRUN || fail "⑤批件零竟升压过（EARLIEST=0 无锚误升）"

# ── T-0437 分面判据：双面批（img=oci=c0）+挡件挂面标 ─────────────
# ⑥红桩——img 面挡 oci 面净=oci 放行：在飞 T-0900 web 件（img 单面）→
#   应发 faces=oci、冻结面=image、reason 携面标 在飞:T-0900(img)。
printf 'o-worker-T-0900.service loaded active running o-worker-T-0900.service\n' > "$UNIT_CTL"
: > "$DMQ_CTRL"; : > "$ESC_CALLS"
rc=0; out=$(run "$C0" "$C0") || rc=$?
[ "$rc" -eq 0 ] || fail "⑥rc=$rc≠0: $out"
[ "$(state "$out")" = 应发 ] || fail "⑥state≠应发: $out"
[ "$(faces "$out")" = oci ] || fail "⑥faces≠oci（img 挡件漏挂面）: $out"
grep -q "分面发窗" <<<"$out" || fail "⑥reason 缺分面发窗标: $out"
grep -q "冻结面=image" <<<"$out" || fail "⑥reason 缺冻结面: $out"
grep -q "放行面=oci" <<<"$out" || fail "⑥reason 缺放行面: $out"
grep -q "在飞:T-0900(img)" <<<"$out" || fail "⑥reason 缺面标挡件: $out"

# ⑦红桩——双面挡件全挡：在飞 T-0901 internal 件（OCI 构建源=双面）→
#   应发面集全冻结→有挡 exit11，面标 (img+oci)。
printf 'o-worker-T-0901.service loaded active running o-worker-T-0901.service\n' > "$UNIT_CTL"
: > "$ESC_CALLS"
rc=0; out=$(run "$C0" "$C0") || rc=$?
[ "$rc" -eq 11 ] || fail "⑦rc=$rc≠11: $out"
[ "$(state "$out")" = 有挡 ] || fail "⑦state≠有挡: $out"
grep -q "在飞:T-0901(img+oci)" <<<"$out" || fail "⑦reason 缺双面挡标: $out"
grep -q "全被挡" <<<"$out" || fail "⑦reason 缺全挡判词: $out"

# ⑧红桩——支树俱无在飞件=不可判双挡（现行语义不退化）：T-0902 无支无树。
printf 'o-worker-T-0902.service loaded active running o-worker-T-0902.service\n' > "$UNIT_CTL"
: > "$ESC_CALLS"
rc=0; out=$(run "$C0" "$C0") || rc=$?
[ "$rc" -eq 11 ] || fail "⑧rc=$rc≠11: $out"
grep -q "在飞:T-0902(不可判→双挡)" <<<"$out" || fail "⑧reason 缺不可判双挡标: $out"

# ⑨红桩——待裁面查询败=fail-closed 双挡（现行语义不退化）。
: > "$UNIT_CTL"; echo "fail 7" > "$DMQ_CTRL"
: > "$ESC_CALLS"
rc=0; out=$(run "$C0" "$C0") || rc=$?
[ "$rc" -eq 11 ] || fail "⑨rc=$rc≠11: $out"
grep -q "待裁面查询败rc=7(不可判→双挡)" <<<"$out" || fail "⑨reason 缺查询败双挡标: $out"

# ⑩红桩——待裁面挂面标：pending T-0903 web 件（img 单面）→oci 放行。
: > "$ESC_CALLS"; printf 'T-0903\n' > "$DMQ_CTRL"
rc=0; out=$(run "$C0" "$C0") || rc=$?
[ "$rc" -eq 0 ] || fail "⑩rc=$rc≠0: $out"
[ "$(faces "$out")" = oci ] || fail "⑩faces≠oci: $out"
grep -q "待裁:T-0903(img)" <<<"$out" || fail "⑩reason 缺待裁面标: $out"

echo "deploy_window_faces_test 全绿"
