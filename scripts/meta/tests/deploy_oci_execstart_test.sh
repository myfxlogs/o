#!/usr/bin/env bash
# deploy_oci_execstart_test.sh —— T-0418 ExecStart specifier 展桩测：
# ①%h→远端 $HOME ②%u→属主名 ③未知 specifier 保守 die 报原值
# ④字面路径回归 ⑤env 包装形保守 die（顺修面） ⑥%% 字面百分号不误伤。
# 桩面：O_OCI_SSH 指桩（远端 cmd eval 落 TMP fixture——systemctl/id/HOME
# 全桩不碰真盒）；go 桩过 arm64 build 段；TMP git 仓+ALLOW_UNPUSHED 过出处闸。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT

# ── 桩仓（出处闸过面：干净树 + O_DEPLOY_ALLOW_UNPUSHED 免 origin/master）──
REPO="$TMP/repo"; mkdir -p "$REPO/scripts" "$REPO/deploy"
cp "$ROOT/scripts/deploy_oci.sh" "$REPO/scripts/"
cp "$ROOT/deploy/oci-env-contract.tsv" "$REPO/deploy/"
git -C "$REPO" init -q
git -C "$REPO" -c user.name=t -c user.email=t@t add -A
git -C "$REPO" -c user.name=t -c user.email=t@t commit -qm init

# ── ssh 桩：末参=远端 cmd，eval 落本地 fixture（systemctl/id/HOME 桩面）──
STUB_FRAG="$TMP/o-collecto.service"
STUB_HOME=/home/ubuntu
STUB_USER=ubuntu
STUB_ENV='SG_PG_DSN=postgres://x'
export STUB_FRAG STUB_HOME STUB_USER STUB_ENV
cat > "$TMP/ssh" <<'EOS'
#!/usr/bin/env bash
cmd="${!#}"
systemctl() {
	local prop="" u=""
	while [ $# -gt 0 ]; do case "$1" in
		show) u=$2; shift 2;; -p) prop=$2; shift 2;; *) shift;; esac; done
	case "$prop" in
		FragmentPath) printf '%s\n' "$STUB_FRAG";;
		Environment) printf '%s\n' "$STUB_ENV";;
		EnvironmentFiles) printf '\n';;
	esac
}
id() { printf '%s\n' "$STUB_USER"; }
HOME="$STUB_HOME"
eval "$cmd"
EOS
chmod +x "$TMP/ssh"

# ── go 桩：-o 落空件过 build 段（DRY 不走 scp/sha 对拍）──
mkdir "$TMP/bin"
cat > "$TMP/bin/go" <<'EOS'
#!/usr/bin/env bash
out=""
while [ $# -gt 0 ]; do case "$1" in -o) out=$2; break;; *) shift;; esac; done
[ -n "$out" ] && : > "$out"
EOS
chmod +x "$TMP/bin/go"
export PATH="$TMP/bin:$PATH"

unit() { printf '[Service]\nExecStart=%s\n' "$1" > "$STUB_FRAG"; }
run() {
	( cd "$REPO" && O_DEPLOY_ALLOW_UNPUSHED=1 O_OCI_SSH="$TMP/ssh" DRY_RUN=1 \
		bash scripts/deploy_oci.sh collecto ) 2>&1
}

# ① %h 形——o-collecto 实证病灶面
unit '%h/o/collector --foo'
out=$(run); grep -q '环约对账绿' <<<"$out" || { echo "①%h 形未过预检: $out" >&2; exit 1; }

# ② %u 形
unit '/home/%u/o/collector'
out=$(run); grep -q '环约对账绿' <<<"$out" || { echo "②%u 形未过预检: $out" >&2; exit 1; }

# ③ 未知 specifier——fail-closed die 显报原值不静默
unit '%t/o/collector'
if out=$(run); then echo "③未知 specifier 竟过" >&2; exit 1; fi
grep -q '未识别 specifier %t' <<<"$out" || { echo "③die 辞不达意: $out" >&2; exit 1; }
grep -q '%t/o/collector' <<<"$out" || { echo "③未报原值: $out" >&2; exit 1; }

# ④ 字面绝对路径回归（原语义面）
unit '/home/ubuntu/o/collector --flag'
out=$(run); grep -q '环约对账绿' <<<"$out" || { echo "④字面路径回归炸: $out" >&2; exit 1; }

# ⑤ env 包装形——保守 die（钦定顺修面：多命令/包装不猜）
unit '/usr/bin/env FOO=1 /home/ubuntu/o/collector'
if out=$(run); then echo "⑤env 包装竟过" >&2; exit 1; fi
grep -q '包装形' <<<"$out" || { echo "⑤未走包装 die: $out" >&2; exit 1; }

# ⑥ %% 字面百分号——不误伤（systemd 转义面）
unit '/home/ubuntu/o/collector --name=100%%cpu'
out=$(run); grep -q '环约对账绿' <<<"$out" || { echo "⑥%% 字面误伤: $out" >&2; exit 1; }

echo "deploy_oci_execstart_test 全绿"
