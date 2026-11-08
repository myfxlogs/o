#!/usr/bin/env bash
# dm_autoapprove_test.sh —— GW-001 段C2/C3 机批闸桩测：psql/ops_esc/
# manual_lane 全桩。面：①册内签名→机批已批+范围钦定 sed ②册值径∩负目录
# →零批 fail-closed rc1 升件（判据③机检断言）③负目录表缺→同律
# ④起草滞留阈 2h 边界（24h→2h）⑤已批 7 天未动→回候选草案行+升件一次
# （批件保鲜水位律）⑥保鲜扫描败滑行不炸。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/repo/docs/tasks"

# ── psql 桩：查询按词形分流 ──
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
  *"k||E'\t'||x"*)
    printf '%s\n' "$query" >> "$SQL_LOG" # 负目录扫描句也落档（INSERT 断言共用面）
    printf '%b\n' "${NEG_ROWS:-}" ;;
  *"string_agg(k"*) printf '%b\n' "${REG_KEYS:-}" ;;
  *"string_agg(x"*) printf '%b\n' "${REG_VALUES:-}" ;;
  *"extract(epoch from now()-max(ts))"*) printf '%s\n' "${STALE_AGE:-0}" ;; # 先于 event='起草'（滞留龄查询同含该词形）
  *"event='起草'"*) printf '%b\n' "${CANDS:-}" ;;
  *"候选滞留:"*) printf '%s\n' "${STALE_DUP:-}" ;;
  *"WITH stale AS"*)
    printf '%s\n' "$query" >> "$SQL_LOG"
    [ -n "${PSQL_BREAK:-}" ] && exit 1
    printf '%b\n' "${BATCH_STALES:-}" ;;
  *"批件超期:"*) printf '%s\n' "${BATCH_DUP:-}" ;;
  *"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$SQL_LOG" ;;
  *) : ;;
esac
EOF
cat > "$TMP/bin/opsesc.sh" <<'EOF'
#!/usr/bin/env bash
printf '%s|%s\n' "$5" "$6" >> "$ESC_LOG"
EOF
cat > "$TMP/bin/manual-lane" <<'EOF'
#!/usr/bin/env bash
exit "${MANUAL_RC:-1}"
EOF
chmod +x "$TMP/bin/"*

# ── 仓桩：候选票件（病灶签名行+范围节骨架行——sed 钦定面） ──
REPO="$TMP/repo"; cd "$REPO"
git init -q && git config user.email t@t && git config user.name t
cat > docs/tasks/T-9001-obs-x.md <<'EOF'
# T-9001 观测盲桩票

- 优先级：P2（起草默认档——DM 裁时可调）

病灶签名：`观测盲:x`（kind=观测盲，key=x）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 明确文件路径或目录清单（批准前必填——T-0413 派单依赖闸据此对拍）；
EOF
cat > docs/tasks/T-9002-execsig.md <<'EOF'
# T-9002 执行面签名桩票

病灶签名：`观测盲:face:table:ops_seat_card`（kind=观测盲，key=face:table:ops_seat_card）

## 范围（起草骨架）

1. 明确文件路径或目录清单（批准前必填——T-0413 派单依赖闸据此对拍）；
EOF
git add -A && git commit -qm init

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export FIN_DSN=fake
export REPO="$REPO"
export OPS_ESC_SH="$TMP/bin/opsesc.sh"
export O_AUTOAPPROVE_NEGCAT="$ROOT/scripts/meta/autoapprove_negative_catalog.tsv"
export MANUAL_RC=1
export SQL_LOG="$TMP/sql.log"; : > "$SQL_LOG"
export ESC_LOG="$TMP/esc.log"; : > "$ESC_LOG"
AA="$ROOT/scripts/meta/dm_autoapprove.sh"

# ── ① 册内签名→机批：已批行+范围节钦定 sed 落面 ──
: > "$SQL_LOG"; : > "$ESC_LOG"
REG_KEYS='观测盲:x' REG_VALUES='internal/research/sigdist' CANDS='T-9001' \
  bash "$AA" >"$TMP/o1.log" 2>&1
grep -q "'CEO','已批','T-9001'" "$SQL_LOG" || { echo "FAIL ①已批行未落"; cat "$SQL_LOG"; exit 1; }
grep -q '机批类册钦定文件径：internal/research/sigdist' docs/tasks/T-9001-obs-x.md \
  || { echo "FAIL ①范围钦定 sed 未落"; cat docs/tasks/T-9001-obs-x.md; exit 1; }
[ ! -s "$ESC_LOG" ] || { echo "FAIL ①误升件"; cat "$ESC_LOG"; exit 1; }
echo "ok ①册内签名机批已批+钦定 sed"

# ── ② 判据③：册值径∩负目录→零批 fail-closed rc1 升件 ──
: > "$SQL_LOG"; : > "$ESC_LOG"
REG_KEYS='观测盲:x' NEG_ROWS='观测盲:x\tinternal/trading/okxexec' CANDS='T-9001' \
  bash "$AA" >"$TMP/o2.log" 2>&1 && { echo "FAIL ②负目录命中仍 rc0"; exit 1; } || rc=$?
[ "$rc" -eq 1 ] || { echo "FAIL ②rc=$rc 非 1"; exit 1; }
grep -q '机批册负目录冲突' "$ESC_LOG" || { echo "FAIL ②缺负目录升件"; cat "$ESC_LOG"; exit 1; }
grep -q 'internal/trading/okxexec' "$ESC_LOG" || { echo "FAIL ②升件未携冲突径"; cat "$ESC_LOG"; exit 1; }
grep -q "'CEO','已批'" "$SQL_LOG" && { echo "FAIL ②负目录冲突仍机批"; cat "$SQL_LOG"; exit 1; }
echo "ok ②负目录命中零批 fail-closed（钱径面机批断言拒）"

# ── ③ 负目录表缺→同律 fail-closed ──
: > "$ESC_LOG"
O_AUTOAPPROVE_NEGCAT=/nonexistent/negcat.tsv bash "$AA" >"$TMP/o3.log" 2>&1 \
  && { echo "FAIL ③表缺仍 rc0"; exit 1; } || rc=$?
[ "$rc" -eq 1 ] || { echo "FAIL ③rc=$rc 非 1"; exit 1; }
grep -q '机批册负目录表缺' "$ESC_LOG" || { echo "FAIL ③缺表升件"; cat "$ESC_LOG"; exit 1; }
echo "ok ③负目录表缺 fail-closed"

# ── ④ 起草滞留阈 2h（GW-001 段C2 24h→2h）：册外签名滞留升件边界 ──
: > "$ESC_LOG"
REG_KEYS='观测盲:x' REG_VALUES='' CANDS='T-9001' STALE_AGE=7199 \
  bash "$AA" >"$TMP/o4.log" 2>&1
[ ! -s "$ESC_LOG" ] || { echo "FAIL ④阈下误升件"; cat "$ESC_LOG"; exit 1; }
: > "$ESC_LOG"
REG_KEYS='观测盲:x' REG_VALUES='' CANDS='T-9001' STALE_AGE=7201 \
  bash "$AA" >>"$TMP/o4.log" 2>&1
grep -q '候选滞留:T-9001' "$ESC_LOG" || { echo "FAIL ④阈上未升件"; cat "$ESC_LOG"; exit 1; }
# 查重：滞留升件已行即不再升
: > "$ESC_LOG"
REG_KEYS='观测盲:x' REG_VALUES='' CANDS='T-9001' STALE_AGE=7201 STALE_DUP=1 \
  bash "$AA" >>"$TMP/o4.log" 2>&1
[ ! -s "$ESC_LOG" ] || { echo "FAIL ④查重失效再升"; cat "$ESC_LOG"; exit 1; }
echo "ok ④起草滞留阈 2h 边界+升件一次去重"

# ── ⑤ 已批 7 天未动→回候选草案行+升件一次（批件保鲜） ──
: > "$SQL_LOG"; : > "$ESC_LOG"
CANDS='' BATCH_STALES='T-9003' bash "$AA" >"$TMP/o5.log" 2>&1
grep -q "'CTO','草案',s.ticket" "$SQL_LOG" \
  || { echo "FAIL ⑤回候选草案行未落"; cat "$SQL_LOG"; exit 1; }
grep -q '批件超期:T-9003' "$ESC_LOG" \
  || { echo "FAIL ⑤RETURNING 票未进升件链"; cat "$ESC_LOG"; exit 1; }
# 同票超期升件已行→升件去重，回候选行照常（生命周期动作与通知分家）
: > "$SQL_LOG"; : > "$ESC_LOG"
CANDS='' BATCH_STALES='T-9003' BATCH_DUP=1 bash "$AA" >>"$TMP/o5.log" 2>&1
grep -q "'CTO','草案',s.ticket" "$SQL_LOG" \
  || { echo "FAIL ⑤去重面回候选行未落"; cat "$SQL_LOG"; exit 1; }
[ ! -s "$ESC_LOG" ] || { echo "FAIL ⑤超期升件未去重"; cat "$ESC_LOG"; exit 1; }
echo "ok ⑤已批 7 天回候选+升件一次（去重不阻回候选）"

# ── ⑥ 保鲜扫描败→升件滑行 rc0（尽力而为不炸拍壳） ──
: > "$SQL_LOG"; : > "$ESC_LOG"
CANDS='' PSQL_BREAK=with-stale bash "$AA" >"$TMP/o6.log" 2>&1 \
  || { echo "FAIL ⑥扫描败非 rc0"; exit 1; }
grep -q '批件保鲜扫描败' "$ESC_LOG" || { echo "FAIL ⑥缺滑行升件"; cat "$ESC_LOG"; exit 1; }
echo "ok ⑥保鲜扫描败滑行不炸"

echo "dm autoapprove segc tests PASS"
