#!/usr/bin/env bash
# kb_recall.sh —— 经验库首阶召回件（提案-KB向量检索-经验库召回面 首阶②）：
# 输入=票面文本（文件路径；`-` 或缺省=stdin），输出=top-K 条目摘要行
# （stdout，一行一条：`- [lib] id — display`）。
# 纯 keywords/scope/display+text 字段机械匹配，零模型零新设施；检索物料
# （docs/*/检索物料.tsv，kbgen gen 再生物）缺席/全零命中=空输出 rc0——
# fail-open：召回是提示词增强非闸件，装配面零段照跑（dispatch_order 同律）。
# 匹配面（全机械可解释）：id 字面 +50；keywords \x1f 词项（多词=AND）+4；
# scope 路径项双向前缀 +3／非路径整串子串 +2；display+text 拉丁词元 +1×3；
# CJK bigram 共享（df≤30 稀有对才计）+1×4——CJK 无分词，bigram 是首阶
# 机械上限，命中率不够再开嵌入阶（提案分阶钦定）。
# 用法: kb_recall.sh <票面文件|-> [K]    env: O_KB_RECALL_REPO=仓根
#   （缺省=脚本上溯两级——scripts/meta/→repo root）、O_KB_RECALL_K=默认K
set -u

IN=${1:--}
K=${2:-${O_KB_RECALL_K:-3}}
case "$K" in ''|*[!0-9]*) K=3 ;; esac
[ "$K" -gt 10 ] && K=10

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=${O_KB_RECALL_REPO:-$(cd "$SCRIPT_DIR/../.." && pwd)}

CF=$(mktemp "${TMPDIR:-/tmp}/kb-recall-corpus.XXXXXX") || exit 0 # tolerate: mktemp 败=召回缺席 fail-open
trap 'rm -f -- "$CF"' EXIT
if [ "$IN" = "-" ]; then
	cat > "$CF" 2>/dev/null || : # tolerate: stdin 读败=空召回 fail-open
else
	cat -- "$IN" > "$CF" 2>/dev/null || : # tolerate: 票件读败=空召回 fail-open
fi
[ -s "$CF" ] || exit 0

# 物料自愈：检索物料=gitignored kbgen 再生物——缺席时经 kbgen 正径重生
# （gen 幂等：writeIfChanged 同内容不写、原子换名防并发半截读；索引面同跑
# 零漂移）。再生败/无 go 径=该库零召回 fail-open——召回不依赖物料先在场。
for pair in "工程经验库:gen" "经济经验库:gen-econ"; do
	lib=${pair%%:*}; sub=${pair##*:}
	[ -s "$REPO/docs/$lib/检索物料.tsv" ] && continue
	[ -f "$REPO/tools/kbgen/main.go" ] || continue
	# cwd 钉 tools/kbgen——与 pre-commit 同径（gen-econ 的 econDirConst=
	# ../../docs/经济经验库 是相对 tools/kbgen 的锚，仓根跑会指飞）。
	( cd "$REPO/tools/kbgen" && go run . "$sub" >/dev/null 2>&1 ) || : # tolerate: 再生败=该库零召回 fail-open
done

python3 - "$K" "$REPO" "$CF" <<'PY' || true # tolerate: python 缺席/评分败=零召回 fail-open
import sys, os, re

K = int(sys.argv[1])
REPO = sys.argv[2]
corpus = open(sys.argv[3], encoding="utf-8", errors="replace").read()
clc = corpus.lower()

MATS = [os.path.join(REPO, "docs", lib, "检索物料.tsv")
        for lib in ("工程经验库", "经济经验库")]

STOP = set("""the and for with from that this https? www com org net into over under
per via not all any can will would should could than then them they have has had
are was were been being does did done each more most other some such only also
just like well back even still way may might must shall own same too very use
used using one two see etc non""".split())

# 票面路径集：仓根已知域首段的 / 词元（o/ 模块前缀归一再收一份）。
TOP = ("internal", "scripts", "docs", "web", "tools", "migrations", "proto",
       "deploy", "skills", "cmd", "bin", "tests", ".github", ".devin")
PATH_RE = re.compile(r"(?<![\w./~-])((?:" + "|".join(re.escape(t) for t in TOP) +
                     r")/[\w./~%\-一-鿿]+)")
tpaths = set()
for m in PATH_RE.finditer(corpus):
    p = m.group(1).rstrip("/")
    if p:
        tpaths.add(p)
        if p.startswith("o/"):
            tpaths.add(p[2:])

def path_hit(item):
    for p in tpaths:
        if p == item or p.startswith(item + "/") or item.startswith(p + "/"):
            return True
    return False

CJK_RE = re.compile(r"[一-鿿]+")

def cjk_bigrams(text):
    out = set()
    for r in CJK_RE.findall(text):
        for i in range(len(r) - 1):
            out.add(r[i:i + 2])
    return out

cbg = cjk_bigrams(corpus)

# contains：词项命中判——纯拉丁短词（≤5 字符）走词边界，防 ci⊂decision、
# git⊂digit、meta⊂metadata 类子串误中；CJK/长词/含界符词走子串（CJK 无
# 词边界可言，check_all 类整词本就带界符）。
def contains(term):
    t = term.lower()
    if t.isascii() and len(t) <= 5:
        return re.search(r"(?<![a-z0-9])" + re.escape(t) + r"(?![a-z0-9])", clc) is not None
    return t in clc

rows = []
for mat in MATS:
    if not os.path.isfile(mat):
        continue
    for ln in open(mat, encoding="utf-8", errors="replace"):
        if ln.startswith("#") or not ln.strip():
            continue
        f = ln.rstrip("\n").split("\t")
        f += [""] * (10 - len(f))
        rows.append(f[:10])

# df 预统计：bigram 出现于多少条目（幂等集——同条目多字段不重复计）。
df = {}
for f in rows:
    lib, _id, _k, _s, _r, disp, text, kw, scope, _sk = f
    hay = " ".join([kw.replace("\x1f", " "), scope.replace("\x1f", " "), disp, text])
    for bg in cjk_bigrams(hay):
        df[bg] = df.get(bg, 0) + 1

DF_MAX = 30  # 稀有 bigram 阈——常用双字（的/了/检查 族）df 超阈不计分

LAT_RE = re.compile(r"[A-Za-z][A-Za-z0-9_.\-]{2,}")
scored = []
for f in rows:
    lib, _id, _k, _s, _r, disp, text, kw, scope, _sk = f
    score = 0
    if _id and _id.lower() in clc:
        score += 50
    if kw:
        for item in kw.split("\x1f"):
            parts = [p for p in item.split() if p]
            if parts and all(contains(p) for p in parts):
                score += 4
    if scope:
        for item in scope.split("\x1f"):
            if not item:
                continue
            if "/" in item:
                if path_hit(item):
                    score += 3
            elif contains(item):
                score += 2
    # 拉丁词元低权面（display+text，封顶 +3）
    nlat = 0
    for tok in set(LAT_RE.findall(disp + " " + text)):
        t = tok.lower()
        if t not in STOP and contains(t):
            nlat += 1
            if nlat >= 3:
                break
    score += nlat
    # CJK bigram 共享面（稀有对才计，封顶 +4）
    hay = " ".join([kw.replace("\x1f", " "), disp, text])
    nbg = 0
    for bg in cjk_bigrams(hay) & cbg:
        if df.get(bg, 0) <= DF_MAX:
            nbg += 1
            if nbg >= 4:
                break
    score += nbg
    if score > 0:
        scored.append((score, lib, _id, disp))

scored.sort(key=lambda r: (-r[0], r[1], r[2]))
for score, lib, _id, disp in scored[:K]:
    print("- [%s] %s — %s" % (lib, _id, disp))
PY
exit 0
