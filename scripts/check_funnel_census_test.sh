#!/usr/bin/env bash
# check_funnel_census_test.sh —— 家族普查闸桩测（工单O-41372 验收锚）：
# 夹具根 FUNNEL_CENSUS_ROOT 注入面——①全量声明绿径；②假族漏声明行=红；
# ③terminal=live 族 drill_shape=none=红；④paper_only note 空=红；
# ⑤manifest 越册行=红。fixture 无 go.mod→语义探针段按设计跳过（静态面全覆）。
# 用法：bash scripts/check_funnel_census_test.sh
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
GATE="$HERE/check_funnel_census.sh"
T=$(mktemp -d /tmp/funnel-census-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

# 夹具根骨架：execfam.go set 块 + grammar.go Template 字段 + migrations INSERT
fixture() {
  local d="$1"
  mkdir -p "$d/internal/platform/execfam" "$d/internal/research/grammar" \
           "$d/internal/portfolio/lifecycle" "$d/migrations"
  cat > "$d/internal/platform/execfam/execfam.go" <<'EOF'
package execfam
var set = map[string]bool{
	"fam_a": true, "fam_b": true, "fam_vol": true,
}
var fundingHedge = map[string]bool{"fam_a": true}
EOF
  cat > "$d/internal/research/grammar/grammar.go" <<'EOF'
package grammar
type Family struct{ Template string }
var _ = []Family{{Template: "fam_b"}, {Template: "fam_gram"}}
EOF
  cat > "$d/migrations/md_9999_fixture.sql" <<'EOF'
INSERT INTO fin_opp_template(template,venue,params) VALUES
  ('fam_mig','okx','{}')
ON CONFLICT DO NOTHING;
EOF
}

manifest_row() { printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" "$5" "$6"; }

write_manifest() { # $1=dir $2..=rows
  local d="$1"; shift
  {
    echo "family	terminal	alive_check	drill_shape	exec_venue	note"
    for r in "$@"; do echo "$r"; done
  } > "$d/internal/platform/execfam/family_manifest.tsv"
}

run_gate() { # $1=dir → rc
  FUNNEL_CENSUS_ROOT="$1" bash "$GATE" >/dev/null 2>&1
}

# ── 案① 绿径：并集4族（fam_a/fam_b execfam+fam_b grammar+fam_mig mig+fam_gram
#    grammar-only）全声明合法 → 绿（fixture 无 go.mod 语义段跳过）──
D="$T/green"; fixture "$D"
write_manifest "$D" \
  "$(manifest_row fam_a live fundingEntry okx_two_leg okx 'live可达')" \
  "$(manifest_row fam_b live generic_eval okx_single okx 'live可达')" \
  "$(manifest_row fam_vol paper_only volEntry dcd okx 'paper_only 事由：演习产能未建')" \
  "$(manifest_row fam_gram paper_only none none okx 'paper_only 事由：execfam未册')" \
  "$(manifest_row fam_mig paper_only generic_eval none okx 'paper_only 事由：候选未收养')"
if run_gate "$D"; then ok "案① 全量声明绿径"; else bad "案① 绿径误红"; FUNNEL_CENSUS_ROOT="$D" bash "$GATE" 2>&1 | tail -8; fi

# ── 案② 假族漏声明行=红（union 含 fam_gram 但 manifest 缺）──
D="$T/missing"; fixture "$D"
write_manifest "$D" \
  "$(manifest_row fam_a live fundingEntry okx_two_leg okx 'x')" \
  "$(manifest_row fam_b live generic_eval okx_single okx 'x')" \
  "$(manifest_row fam_vol paper_only volEntry dcd okx 'x')" \
  "$(manifest_row fam_mig paper_only generic_eval none okx 'x')"
if run_gate "$D"; then bad "案② 漏声明行未红"; else
  out=$(FUNNEL_CENSUS_ROOT="$D" bash "$GATE" 2>&1); \
  echo "$out" | grep -q "fam_gram" && ok "案② 假族漏声明行=红" || { bad "案② 红但缺 fam_gram 行名"; echo "$out" | tail -6; }
fi

# ── 案③ terminal=live 族 drill_shape=none=红 ──
D="$T/livenone"; fixture "$D"
write_manifest "$D" \
  "$(manifest_row fam_a live fundingEntry none okx 'x')" \
  "$(manifest_row fam_b live generic_eval okx_single okx 'x')" \
  "$(manifest_row fam_vol paper_only volEntry dcd okx 'x')" \
  "$(manifest_row fam_gram paper_only none none okx 'x')" \
  "$(manifest_row fam_mig paper_only generic_eval none okx 'x')"
if run_gate "$D"; then bad "案③ live+drill none 未红"; else
  out=$(FUNNEL_CENSUS_ROOT="$D" bash "$GATE" 2>&1); \
  echo "$out" | grep -q "terminal=live" && ok "案③ live 族 drill_shape=none=红" || { bad "案③ 红但非 terminal 判词"; echo "$out" | tail -6; }
fi

# ── 案④ paper_only note 空=红（注释律）──
D="$T/nonote"; fixture "$D"
write_manifest "$D" \
  "$(manifest_row fam_a live fundingEntry okx_two_leg okx '')" \
  "$(manifest_row fam_b live generic_eval okx_single okx '')" \
  "$(manifest_row fam_vol paper_only volEntry dcd okx '')" \
  "$(manifest_row fam_gram paper_only none none okx '')" \
  "$(manifest_row fam_mig paper_only generic_eval none okx '')"
if run_gate "$D"; then bad "案④ paper_only 空 note 未红"; else
  out=$(FUNNEL_CENSUS_ROOT="$D" bash "$GATE" 2>&1); \
  echo "$out" | grep -q "注释律" && ok "案④ paper_only 空 note=红" || { bad "案④ 红但非注释律判词"; echo "$out" | tail -6; }
fi

# ── 案⑤ manifest 越册行=红（册外族声明=陈旧漂移）──
D="$T/extra"; fixture "$D"
write_manifest "$D" \
  "$(manifest_row fam_a live fundingEntry okx_two_leg okx 'x')" \
  "$(manifest_row fam_b live generic_eval okx_single okx 'x')" \
  "$(manifest_row fam_vol paper_only volEntry dcd okx 'x')" \
  "$(manifest_row fam_gram paper_only none none okx 'x')" \
  "$(manifest_row fam_mig paper_only generic_eval none okx 'x')" \
  "$(manifest_row fam_ghost paper_only none none okx 'x')"
if run_gate "$D"; then bad "案⑤ 越册行未红"; else
  out=$(FUNNEL_CENSUS_ROOT="$D" bash "$GATE" 2>&1); \
  echo "$out" | grep -q "fam_ghost" && ok "案⑤ manifest 越册行=红" || { bad "案⑤ 红但缺 fam_ghost 行名"; echo "$out" | tail -6; }
fi

echo "── funnel-census 桩测：$PASS 绿 / $FAILN 红 ──"
[ "$FAILN" -eq 0 ]
