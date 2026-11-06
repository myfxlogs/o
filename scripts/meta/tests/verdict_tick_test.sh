#!/usr/bin/env bash
# verdict_tick_test.sh —— T-0385 裁域扩水印闸单测：econ+eng pending 合闸
# （eng-only pending 也烧拍；eng 表缺=败兜底算零不炸；双零=skip 不烧）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" /var/tmp/o-verdict

# 桩 psql：ECON_N/ENG_N/ENG_FAIL env 控两表 pending 答数；其余查询哑答。
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
query=${*: -1}
case "$query" in
	*"FROM fin_econ_lesson WHERE status='pending'"*) echo "${ECON_N:-0}" ;;
	*"FROM fin_eng_lesson WHERE status='pending'"*)
		[ "${ENG_FAIL:-0}" = 1 ] && exit 1
		echo "${ENG_N:-0}" ;;
	*"INSERT INTO ops_headless"*) : ;;
	*) echo "unexpected psql: $query" >&2; exit 2 ;;
esac
EOF
# 桩 devin：被调即记一行（烧拍=水印放行实证）。
cat > "$TMP/bin/devin" <<'EOF'
#!/usr/bin/env bash
echo called >> "$DEVIN_CALLS"
EOF
chmod +x "$TMP/bin/psql" "$TMP/bin/devin"
printf 'FIN_PG_DSN=fake-dsn\n' > "$TMP/env"
export DEVIN_CALLS="$TMP/calls" DEVIN_BIN="$TMP/bin/devin"
VT="$ROOT/scripts/meta/verdict_tick.sh"
run() { # $1=econ $2=eng $3=eng_fail
	ECON_N="$1" ENG_N="$2" ENG_FAIL="${3:-0}" PATH="$TMP/bin:$PATH" \
		VERDICT_ENV_FILE="$TMP/env" "$VT" >/dev/null 2>&1 || true
}

# ① 双零=skip 不烧（空拍心跳照写）。
run 0 0 0
[ ! -s "$DEVIN_CALLS" ]

# ② eng-only pending=2（econ=0）→烧拍（T-0385 裁域扩实证）。
run 0 2 0
[ "$(wc -l < "$DEVIN_CALLS")" -eq 1 ]

# ③ eng 表缺（查询败）→兜底算零不炸；econ=0 → 不烧。
run 0 0 1
[ "$(wc -l < "$DEVIN_CALLS")" -eq 1 ]

# ④ econ pending=1 照旧烧（原径零回归）。
run 1 0 1
[ "$(wc -l < "$DEVIN_CALLS")" -eq 2 ]

echo "verdict_tick watermark econ+eng PASS"
