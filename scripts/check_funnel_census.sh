#!/usr/bin/env bash
# check_funnel_census.sh —— 家族能力声明册v0+漏斗普查闸（工单O-41372——
# migrate_table 手工登记漂移的族级同型根修：族语义从 N 份手工 switch 搬进
# 数据册，普查闸钉死回漂）。本闸=家族语义单一源的外部对拍——只查一致性
# 不算语义（禁在闸内重实现判据：LegShapeOf/alive 路由等语义判词一律调
# 真实函数/读真实 switch 面，闸自身零判据数学）。
#
# 断言面（spec 钦定）：
#   ① 启用族并集（execfam.set ∪ grammar.Families ∪ migrations
#      fin_opp_template 种子行）每族 manifest 必有行 ∧ manifest 无越册行
#      （双向=单一源——多行=陈旧声明漂移同红）；
#   ② 列值合法（terminal/alive_check/drill_shape/exec_venue 四枚举域）+
#      terminal=live 族四列无 none + paper_only 行 note 非空（注释律）；
#   ③ alive_check 与 signalAlive 实路由一致（volEntry 显式臂→volEntry｜
#      execfam.FundingHedge 族册→fundingEntry｜execfam.Has=false→none｜
#      余→generic_eval——O-41370 后 default 泛化 eval 判）；
#   ④ drill_shape 与 excell.LegShapeOf+Drillable 实况一致（go probe 真
#      函数直调——FUNNEL_CENSUS_ROOT 无 go.mod 的 fixture 根跳过此段）；
#   ⑤ terminal=live ⇒ execfam.Has=true（提案路径物理前提——live 可达族
#      必在册否则声明与现实必有一错）。
#
# 用法：bash scripts/check_funnel_census.sh
#   FUNNEL_CENSUS_ROOT=<夹具根> 桩测注入面（缺省=仓根）。
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/.." && pwd)
ROOT=${FUNNEL_CENSUS_ROOT:-$REPO}
MANIFEST="$ROOT/internal/platform/execfam/family_manifest.tsv"

FAIL=0
bad() { echo "FAIL funnel-census: $1" >&2; FAIL=1; }
info() { echo "  .. $1"; }

[ -f "$MANIFEST" ] || { echo "FAIL funnel-census: manifest 缺席 $MANIFEST" >&2; exit 1; }

WORK=$(mktemp -d /tmp/funnel-census-XXXXXX)
PROBE_DIR=""
cleanup() { rm -rf "$WORK"; [ -n "$PROBE_DIR" ] && rm -rf "$PROBE_DIR"; }
trap cleanup EXIT

# ── ① manifest 解析+枚举域+注释律+terminal 约束（python 集中判）──
python3 - "$MANIFEST" "$WORK" <<'PYEOF'
import sys
path, work = sys.argv[1], sys.argv[2]
TERM = {"live", "paper_only"}
ALIVE = {"generic_eval", "volEntry", "fundingEntry", "none"}
DRILL = {"okx_two_leg", "okx_single", "cfd_single", "dcd", "bounded", "none"}
VENUE = {"okx", "exness", "xmglobal", "none"}
rows, seen, errs = [], set(), []
with open(path, encoding="utf-8") as f:
    for ln, raw in enumerate(f, 1):
        line = raw.rstrip("\n")
        if not line or line.startswith("#") or line.startswith("family\t"):
            continue
        cols = line.split("\t")
        if len(cols) != 6:
            errs.append(f"行{ln} 列数={len(cols)}≠6")
            continue
        fam, term, alive, drill, venue, note = cols
        if fam in seen:
            errs.append(f"行{ln} 族键重复: {fam}")
        seen.add(fam)
        if term not in TERM: errs.append(f"行{ln} {fam} terminal 非法: {term}")
        if alive not in ALIVE: errs.append(f"行{ln} {fam} alive_check 非法: {alive}")
        if drill not in DRILL: errs.append(f"行{ln} {fam} drill_shape 非法: {drill}")
        if venue not in VENUE: errs.append(f"行{ln} {fam} exec_venue 非法: {venue}")
        if term == "live" and "none" in (alive, drill, venue):
            errs.append(f"行{ln} {fam} terminal=live 但四列含 none（alive={alive} drill={drill} venue={venue}）")
        if term == "paper_only" and not note.strip():
            errs.append(f"行{ln} {fam} paper_only 但 note 空——注释律违例（为什么不进live必填）")
        rows.append((fam, term, alive, drill, venue, note))
with open(f"{work}/manifest.tsv", "w", encoding="utf-8") as f:
    for r in rows:
        f.write("\t".join(r) + "\n")
with open(f"{work}/manifest.errs", "w", encoding="utf-8") as f:
    f.write("\n".join(errs) + ("\n" if errs else ""))
PYEOF
if [ -s "$WORK/manifest.errs" ]; then
  while IFS= read -r e; do [ -n "$e" ] && bad "manifest: $e"; done < "$WORK/manifest.errs"
fi
MROWS=$(wc -l < "$WORK/manifest.tsv" | tr -d ' ')
[ "$MROWS" -gt 0 ] || bad "manifest 有效行=0——册面空=闸无对象"

# ── ② 启用族并集枚举（execfam.set ∪ grammar.Families ∪ mig 种子）──
# E：execfam.go set 块 "name": true
sed -n '/var set = map\[string\]bool{/,/^}/p' \
  "$ROOT/internal/platform/execfam/execfam.go" 2>/dev/null \
  | grep -oE '"[a-zA-Z0-9_-]+":[[:space:]]*true' \
  | sed -E 's/^"([^"]+)".*/\1/' | sort -u > "$WORK/u_execfam.txt"
# G：grammar.go Template: "name"
grep -oE 'Template:[[:space:]]*"[a-zA-Z0-9_-]+"' \
  "$ROOT/internal/research/grammar/grammar.go" 2>/dev/null \
  | sed -E 's/.*"([^"]+)"/\1/' | sort -u > "$WORK/u_grammar.txt"
# M：migrations INSERT INTO fin_opp_template 首列模板值（python 全形解析——
#    VALUES 组/select-from-values 两形都收，列清单定位 template 位）
python3 - "$ROOT/migrations" "$WORK/u_mig.txt" <<'PYEOF'
import re, sys, glob, os
mdir, out = sys.argv[1], sys.argv[2]
tmpls = set()
for f in sorted(glob.glob(os.path.join(mdir, "*.sql"))):
    src = open(f, encoding="utf-8", errors="replace").read()
    for m in re.finditer(r"INSERT\s+INTO\s+fin_opp_template\b(.*?);", src, re.S | re.I):
        block = m.group(1)
        cm = re.match(r"\s*\(([^)]*)\)", block)
        ti = None
        if cm:
            cols = [c.strip() for c in cm.group(1).split(",")]
            if "template" in cols:
                ti = cols.index("template")
        for t in re.finditer(r"\(([^()]*)\)", block):
            vals = re.findall(r"'((?:[^'\\]|\\.)*)'", t.group(1))
            if not vals:
                continue
            cand = vals[ti] if (ti is not None and len(vals) > ti) else (vals[0] if ti is None else None)
            if cand:
                tmpls.add(cand)
with open(out, "w", encoding="utf-8") as fh:
    fh.write("\n".join(sorted(tmpls)) + ("\n" if tmpls else ""))
PYEOF
cat "$WORK/u_execfam.txt" "$WORK/u_grammar.txt" "$WORK/u_mig.txt" \
  | grep -v '^$' | sort -u > "$WORK/union.txt"
cut -f1 "$WORK/manifest.tsv" | sort -u > "$WORK/manifest_keys.txt"

# 覆盖双向断言（缺行=漏登记；越册=陈旧声明——同红）
comm -23 "$WORK/union.txt" "$WORK/manifest_keys.txt" > "$WORK/missing.txt"
comm -13 "$WORK/union.txt" "$WORK/manifest_keys.txt" > "$WORK/extra.txt"
while IFS= read -r t; do [ -n "$t" ] && bad "启用族缺 manifest 行: $t"; done < "$WORK/missing.txt"
while IFS= read -r t; do [ -n "$t" ] && bad "manifest 越册行（不在启用族并集）: $t"; done < "$WORK/extra.txt"
UN=$(wc -l < "$WORK/union.txt" | tr -d ' ')
NE=$(wc -l < "$WORK/u_execfam.txt" | tr -d ' ')
NG=$(wc -l < "$WORK/u_grammar.txt" | tr -d ' ')
NM=$(wc -l < "$WORK/u_mig.txt" | tr -d ' ')
info "并集=$UN 族（execfam=$NE grammar=$NG mig=$NM）manifest=$MROWS 行"

# ── ③④⑤ 语义对拍（go probe 真函数——fixture 根无 go.mod 跳过）──
if [ ! -f "$ROOT/go.mod" ]; then
  info "FUNNEL_CENSUS_ROOT 无 go.mod——语义探针段跳过（fixture 桩测面）"
else
  PROBE_DIR=$(mktemp -d "$REPO/.funnelcensus_probe.XXXXXX")
  cat > "$PROBE_DIR/main.go" <<'GOEOF'
package main

import (
	"bufio"
	"fmt"
	"os"

	"o/internal/platform/execfam"
	"o/internal/platform/excell"
)

func main() {
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		t := sc.Text()
		if t == "" {
			continue
		}
		shape, _ := excell.LegShapeOf(t)
		fmt.Printf("%s\t%t\t%t\t%t\t%s\t%t\n",
			t, execfam.Has(t), execfam.FundingHedge(t), execfam.PaperOnly(t),
			string(shape), excell.Drillable(t))
	}
}
GOEOF
  if ! (cd "$REPO" && go run "$PROBE_DIR" < "$WORK/union.txt" > "$WORK/probe.tsv" 2>"$WORK/probe.err"); then
    bad "语义探针编译/运行败——闸判据残缺 fail-closed"
    sed 's/^/    probe: /' "$WORK/probe.err" | head -10 >&2
  else
    # signalAlive volEntry 显式臂模板集——读 retire.go 真实 switch 面
    # （case … r.volEntryGate 臂内模板字面量；判据=读码面非重实现）
    python3 - "$ROOT/internal/portfolio/lifecycle/retire.go" "$WORK" <<'PYEOF'
import re, sys
src_path, work = sys.argv[1], sys.argv[2]
src = open(src_path, encoding="utf-8").read()
m = re.search(r"func \(r \*Runner\) signalAlive\(.*?\n\}\n", src, re.S)
body = m.group(0) if m else ""
vol = set()
for cm in re.finditer(r"case\s+(.*?)\n\s*ok, _ := r\.volEntryGate", body, re.S):
    vol.update(re.findall(r'"([a-zA-Z0-9_-]+)"', cm.group(1)))
with open(f"{work}/volarm.txt", "w") as f:
    f.write("\n".join(sorted(vol)) + ("\n" if vol else ""))
PYEOF
    # 期望值合成+比对（probe.tsv: fam has fh po legshape drillable）
    python3 - "$WORK" <<'PYEOF'
import sys
work = sys.argv[1]
probe = {}
for line in open(f"{work}/probe.tsv", encoding="utf-8"):
    p = line.rstrip("\n").split("\t")
    if len(p) == 6:
        probe[p[0]] = p[1:]
volarm = set(x for x in open(f"{work}/volarm.txt").read().split() if x)
# LegShape→manifest drill_shape 映（bounded=grid_bot bot全径仿真族）
SHAPE_MAP = {"okx_two_leg": "okx_two_leg", "okx_single_leg": "okx_single",
             "cfd_single_leg": "cfd_single", "dcd": "dcd",
             "grid_bot": "bounded"}
errs = []
for line in open(f"{work}/manifest.tsv", encoding="utf-8"):
    fam, term, alive, drill, venue, note = line.rstrip("\n").split("\t")
    if fam not in probe:
        errs.append(f"{fam} 不在并集——probe 无语义面（前段并集断言应已红）")
        continue
    has, fh, po, shape, drillable = probe[fam]
    has = has == "true"; fh = fh == "true"; drillable = drillable == "true"
    # alive_check 期望：!Has→none｜vol臂→volEntry｜FundingHedge→fundingEntry｜余→generic_eval
    if not has:
        exp_alive = "none"
    elif fam in volarm:
        exp_alive = "volEntry"
    elif fh:
        exp_alive = "fundingEntry"
    else:
        exp_alive = "generic_eval"
    if alive != exp_alive:
        errs.append(f"{fam} alive_check={alive}≠实路由={exp_alive}（has={has} fh={fh} volarm={fam in volarm}）")
    # drill_shape 期望：Drillable→LegShapeOf 映｜否→none
    exp_drill = SHAPE_MAP.get(shape, "none") if drillable else "none"
    if drill != exp_drill:
        errs.append(f"{fam} drill_shape={drill}≠LegShapeOf/Drillable 实况={exp_drill}（shape={shape} drillable={drillable}）")
    # terminal=live ⇒ execfam.Has 前置（提案路径物理前提）
    if term == "live" and not has:
        errs.append(f"{fam} terminal=live 但 execfam.Has=false——提案路径不存在=live 不可达")
with open(f"{work}/sem.errs", "w", encoding="utf-8") as f:
    f.write("\n".join(errs) + ("\n" if errs else ""))
PYEOF
    if [ -s "$WORK/sem.errs" ]; then
      while IFS= read -r e; do [ -n "$e" ] && bad "语义对拍: $e"; done < "$WORK/sem.errs"
    else
      info "语义对拍全绿（alive_check/drill_shape 与真函数/真路由一致）"
    fi
  fi
fi

if [ "$FAIL" -ne 0 ]; then
  echo "FAIL check_funnel_census（家族声明册与启用族并集/语义面漂移——见上 FAIL 行）" >&2
  exit 1
fi
echo "OK check_funnel_census（家族能力声明册 v0 在册——并集 $UN 族全覆盖+语义面一致）"
