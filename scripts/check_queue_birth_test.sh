#!/usr/bin/env bash
# check_queue_birth_test.sh —— 消费面出生闸桩测（工单O-35484①验收）：
# 夹具树喂 QUEUE_BIRTH_SCAN_ROOT/QUEUE_BIRTH_MIG_DIR/QUEUE_BIRTH_REG——
# ①绿基线：存量 view 全登记 rc=0；②新 view 未登记=红；③新生面缺三件套=红；
# ④幻影登记（登记 view 机扫无）=红；⑤豁免词非「存量」起头=红；
# ⑥字段数≠6=红；⑦新生面三件套齐=绿；⑧类不符=红；⑨登记表缺席=红。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
LINT="$HERE/check_queue_birth.sh"
FAIL=0; PASS=0
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT

ok()  { PASS=$((PASS+1)); echo "PASS $1"; }
bad() { FAIL=$((FAIL+1)); echo "FAIL $1" >&2; }

# 夹具基线：migrations 一存量 view + 登记表对应存量行
mkroot() { # $1=dir
	mkdir -p "$1/migrations" "$1/docs"
	cat > "$1/migrations/fin_0001_a.sql" <<'EOF'
CREATE VIEW old_face AS SELECT 1;
EOF
	cat > "$1/docs/queue_birth_registry.txt" <<'EOF'
# fixture registry
old_face|view|-|-|-|存量普查2026-11-24
EOF
}

run() { # $1=dir → rc
	QUEUE_BIRTH_SCAN_ROOT="$1" bash "$LINT" >/dev/null 2>&1
}

# ①绿基线
D=$T/g1; mkroot "$D"
if run "$D"; then ok "绿基线-存量全登记"; else bad "绿基线-存量全登记"; fi

# ②新 view 未登记=红
D=$T/g2; mkroot "$D"
cat >> "$D/migrations/fin_0002_b.sql" <<'EOF'
CREATE VIEW brand_new_q AS SELECT 1;
EOF
if run "$D"; then bad "新view未登记应红"; else ok "新view未登记=红"; fi

# ③新生面登记但缺三件套（无豁免）=红
D=$T/g3; mkroot "$D"
cat >> "$D/migrations/fin_0002_b.sql" <<'EOF'
CREATE VIEW brand_new_q AS SELECT 1;
EOF
cat >> "$D/docs/queue_birth_registry.txt" <<'EOF'
brand_new_q|view|-|-|-|
EOF
if run "$D"; then bad "缺三件套应红"; else ok "缺三件套=红"; fi

# ④幻影登记=红（登记 view 机扫集无）
D=$T/g4; mkroot "$D"
cat >> "$D/docs/queue_birth_registry.txt" <<'EOF'
ghost_face|view|-|-|-|存量普查2026-11-24
EOF
if run "$D"; then bad "幻影登记应红"; else ok "幻影登记=红"; fi

# ⑤豁免词非「存量」起头=红（新生禁挂豁免——词面闸）
D=$T/g5; mkroot "$D"
cat >> "$D/migrations/fin_0002_b.sql" <<'EOF'
CREATE VIEW sneaky_q AS SELECT 1;
EOF
cat >> "$D/docs/queue_birth_registry.txt" <<'EOF'
sneaky_q|view|-|-|-|新面豁免
EOF
if run "$D"; then bad "豁免词不符应红"; else ok "豁免词不符=红"; fi

# ⑥字段数≠6=红
D=$T/g6; mkroot "$D"
cat >> "$D/docs/queue_birth_registry.txt" <<'EOF'
bad_row|view|-|-|-
EOF
if run "$D"; then bad "字段数不符应红"; else ok "字段数不符=红"; fi

# ⑦新生面三件套齐=绿
D=$T/g7; mkroot "$D"
cat >> "$D/migrations/fin_0002_b.sql" <<'EOF'
CREATE OR REPLACE VIEW brand_new_q AS SELECT 1;
EOF
cat >> "$D/docs/queue_birth_registry.txt" <<'EOF'
brand_new_q|view|fin_0002 filed_ts>24h|esc_scope 折叠|digestspawn→工单|
EOF
if run "$D"; then ok "三件套齐=绿"; else bad "三件套齐应绿"; fi

# ⑧类不符=红
D=$T/g8; mkroot "$D"
cat >> "$D/docs/queue_birth_registry.txt" <<'EOF'
weird_face|widget|-|-|-|存量普查2026-11-24
EOF
if run "$D"; then bad "类不符应红"; else ok "类不符=红"; fi

# ⑨登记表缺席=红
D=$T/g9; mkroot "$D"; rm "$D/docs/queue_birth_registry.txt"
if run "$D"; then bad "登记表缺席应红"; else ok "登记表缺席=红"; fi

# ⑩queue/谓词类非豁免行同样要三件套（钦定面不豁免机扫盲区）
D=$T/g10; mkroot "$D"
cat >> "$D/docs/queue_birth_registry.txt" <<'EOF'
new_queue|queue|龄界24h|同根签名|升件→工单|
EOF
if run "$D"; then ok "queue三件套齐=绿"; else bad "queue三件套齐应绿"; fi

echo "── check_queue_birth_test: PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ]
