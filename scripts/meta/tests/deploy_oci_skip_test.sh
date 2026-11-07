#!/usr/bin/env bash
# deploy_oci_skip_test.sh —— T-0425 件③谱系自检桩测：
# ①diff 全件证外（image_only web）→全臂跳 幂等快退零烧 exit0 不 build
# ②diff 碰 projector 谱系→projector 留 他臂跳（臂级粒度实证）
# ③diff 碰不可判面（未列名 internal/guard）→全臂保守照换
# ④臂 .sha 缺件→不可证空 该臂照换
# 桩面：TMP git 仓+fixture service/.sha；O_OCI_SSH 桩（systemctl/id
# eval 落本地 fixture 不碰真盒）；go 桩过 build 段；ALLOW_UNPUSHED
# 过出处闸；DRY_RUN=1 不走锁/船运。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf -- "$TMP"' EXIT

# ── 桩仓：c0 基件 → 各案 c1 由 mkcase 换树 ───────────────
REPO="$TMP/repo"; mkdir -p "$REPO/scripts" "$REPO/deploy" "$REPO/internal/guard"
cp "$ROOT/scripts/deploy_oci.sh" "$REPO/scripts/"
cp "$ROOT/deploy/oci-env-contract.tsv" "$ROOT/deploy/deploy-faces.tsv" "$REPO/deploy/"
git -C "$REPO" init -q -b master
G() { git -C "$REPO" -c user.name=t -c user.email=t@t "$@"; }
echo a > "$REPO/base.txt"; G add -A; G commit -qm c0; C0=$(G rev-parse HEAD)

# ── ssh 桩：末参=远端 cmd，eval 落本地 fixture（systemctl/id/HOME 桩面）──
STUB_FRAGDIR="$TMP/frags"; STUB_HOME="$TMP/armhome"; STUB_USER=ubuntu
mkdir -p "$STUB_FRAGDIR" "$STUB_HOME/o"
for u in o-collecto:o/collector o-projector:o/projector o-research:o-research/o-research; do
	un=${u%%:*}; ep=${u##*:}
	mkdir -p "$STUB_HOME/$(dirname "$ep")"
	printf '[Service]\nExecStart=%%h/%s\n' "$ep" > "$STUB_FRAGDIR/$un.service"
	printf 'sha=%s\n' "$C0" > "$STUB_HOME/$ep.sha"
done
export STUB_FRAGDIR STUB_HOME STUB_USER
cat > "$TMP/ssh" <<'EOS'
#!/usr/bin/env bash
cmd="${!#}"
systemctl() {
	local prop="" u=""
	while [ $# -gt 0 ]; do case "$1" in
		show) u=$2; shift 2;; -p) prop=$2; shift 2;; *) shift;; esac; done
	case "$prop" in
		FragmentPath) printf '%s\n' "$STUB_FRAGDIR/$u.service";;
		Environment) printf '%s\n' 'SG_PG_DSN=x MD_PG_DSN=x FIN_PG_DSN=x ROSTER_PG_DSN=x';;
		EnvironmentFiles) printf '\n';;
	esac
}
id() { printf '%s\n' "$STUB_USER"; }
HOME="$STUB_HOME"
eval "$cmd"
EOS
chmod +x "$TMP/ssh"

# ── go 桩 ───────────────────────────────────────────────
mkdir "$TMP/bin"
printf '#!/usr/bin/env bash\nout=""\nwhile [ $# -gt 0 ]; do case "$1" in -o) out=$2; break;; *) shift;; esac; done\n[ -n "$out" ] && : > "$out"\n' > "$TMP/bin/go"
chmod +x "$TMP/bin/go"
export PATH="$TMP/bin:$PATH"

run() {
	( cd "$REPO" && O_DEPLOY_ALLOW_UNPUSHED=1 O_OCI_SSH="$TMP/ssh" DRY_RUN=1 \
		bash scripts/deploy_oci.sh "$@" ) 2>&1
}
fail() { echo "❌ $*" >&2; exit 1; }
mkcase() { # mkcase <path> —— c1 落单件提交
	: > "$ESC"; mkdir -p "$REPO/$(dirname "$1")"; echo y > "$REPO/$1"
	G add -A; G commit -qm "case $1" >/dev/null; C1=$(G rev-parse HEAD)
}
ESC="$TMP/.mkcase-sentinel"; : > "$ESC"

# ①diff 全件证外——web/x.ts=image_only→三臂全跳 幂等快退零烧
mkcase web/w.ts
out=$(run) || fail "①rc≠0: $out"
grep -q '全臂谱系 diff 证空——幂等快退零烧' <<<"$out" || fail "①未走快退: $out"
grep -q '本地 arm64 build' <<<"$out" && fail "①证空竟 build: $out"

# ②diff 碰 projector 谱系——projector 留 他臂跳（臂级粒度）
G reset -q --hard "$C0"; mkcase internal/projector/tick.go
out=$(run) || fail "②rc≠0: $out"
grep -q '待换臂: projector' <<<"$out" || fail "②待换臂非独 projector: $out"
grep -q 'o-collecto 谱系 diff 证空' <<<"$out" || fail "②collecto 未跳: $out"

# ③diff 碰不可判面——internal/guard 未列名→全臂保守照换
G reset -q --hard "$C0"; mkcase internal/guard/g.go
out=$(run) || fail "③rc≠0: $out"
grep -q '待换臂: collecto projector research' <<<"$out" || fail "③不可判面未全换: $out"

# ④臂 .sha 缺件——o-research.sha 撤→research 照换 他臂仍跳
G reset -q --hard "$C0"; mkcase web/w2.ts
rm -f "$STUB_HOME/o-research/o-research.sha"
out=$(run) || fail "④rc≠0: $out"
grep -q '待换臂: research' <<<"$out" || fail "④.sha 缺臂未照换: $out"

echo "deploy_oci_skip_test 全绿"
