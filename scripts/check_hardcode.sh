#!/usr/bin/env bash
# check_hardcode.sh —— O-41383 硬编码普查闸（名册真源/缺省单源/谓词
# 收编三面验收式常驻化——劝诫会失效、闸才守得住；同 D-514 import 闸/
# 免疫闸/resident 普查同模具）。
#
# 断言面（违例=红+违例行打印；扫描域=internal/ 非测试 go 件——测试
# 夹具 venue 名=假户标签非名册求解面）：
#   ① CFD venue 字面量 '"exness"'|'"xmglobal"'|''exness''|''xmglobal''
#     出白名单文件集外（名册=fin_venue_roster 唯一真源——ActiveCfdVenue/
#     IsCfdProtoVenue 求解面；cmd/tools 面= fleet/面板兼容域另票面不扫）；
#   ② '-SWAP' 后缀判 HasSuffix.*-SWAP 出 legset.go 外（谓词单源=
#     legset.IsSwapInst/IsUSDTSwapInst）；
#   ③ getF/GetFloat/f( 回退位裸数字字面量（120.0/7200/0.15/0.30/240.0
#     ——缺省值必引 legset.*Def* 常量；粗扫宁误报，白名单钉误报面）；
#   ④ venue/proto 枚举再造件启发式（proto='x' 字面谓词出名册读面宿主
#     外——新 CFD 户名硬编码进判据=枚举第二套苗头）。
#
# 白名单=脚本内嵌册（同 check_import_direction.sh 先例——册即显形决策点，
# 扩册须改本脚本=显形决策）：
#   internal/mt5/**                名册读面宿主域（roster.go/symmatch 等）
#   depenum_mt5venue.go            名册枚举宿主（proto='cfd' 全列读面）
#   grammar/grammar.go             模板族声明面（inert——NeedFace/PrimaryVen
#                                  venue 名=面族需求声明元数据，Existing 族
#                                  不生新行；名册化=grammar Enumerate join
#                                  面扩展另票设计面，O-41383 spec 外不扩）
#   ledger/ledger.go               ccyAllowed roster proto='cfd' 读面宿主
#   oppcensus_facts.go             roster proto='cfd' 读面宿主
#   xvenueswap.go                  已修——保留位防回退（现无字面量）
# ③白名单：dirpaper.go markIv=240.0（spec 钦定留本地——pnl_mark_min 域
#   另键非 lifecycle.dir.* 缺省族）；okxgrid/verdict.go f(...,0.15)
#   （okxgrid.stop_loss_frac 异键域缺省——与 DirStopDefFrac 同值异义
#   禁混用）；legset.go 常量定义位本体。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)" || exit 1

FAIL=0
note() { echo "  违例: $1"; }

scan() { # $1=描述 $2=grep正则 $3=排除正则(白名单,ERE对路径) $4=附加排除ERE(可选)
	local hits
	hits=$(grep -rnE "$2" internal/ --include='*.go' 2>/dev/null \
		| grep -v '_test\.go' \
		| grep -vE ':[0-9]+:[[:space:]]*//' \
		| grep -vE "$3" || true) # tolerate: grep 零命中=rc1——空集即净非吞败（断言面=hits 非空才红）
	if [ -n "${4:-}" ] && [ -n "$hits" ]; then
		hits=$(printf '%s\n' "$hits" | grep -vE "$4" || true) # tolerate: 附加排除零命中=全滤净同上
	fi
	if [ -n "$hits" ]; then
		FAIL=1
		echo "❌ [hardcode] $1"
		echo "$hits" | sed 's/^/    /'
	fi
}

# ① venue 字面量（双引号 Go 串+单引号 SQL 串两形）
VEN_WL='internal/mt5/|internal/portfolio/hunger/depenum_mt5venue\.go|internal/research/grammar/grammar\.go|internal/portfolio/lifecycle/xvenueswap\.go'
scan "CFD venue 字面量（名册求解面外硬编码）" \
	'"exness"|"xmglobal"|'"'"'exness'"'"'|'"'"'xmglobal'"'"'' \
	"$VEN_WL"

# ② -SWAP 谓词平行件
scan "HasSuffix -SWAP 谓词（legset.IsSwapInst 单源外残留）" \
	'HasSuffix\([^)]*,\s*"-SWAP"' \
	'internal/platform/legset/legset\.go'

# ③ 缺省裸数字字面量（getF/GetFloat/f( 回退参+:= 缺省位两形）
NUM_WL='internal/platform/legset/legset\.go|internal/portfolio/lifecycle/dirpaper\.go|internal/trading/okxgrid/verdict\.go'
scan "缺省裸数字字面量（legset.*Def* 常量单源外）" \
	'(GetFloat|\.Get\(|getF|f\()\s*\([^)]*,\s*(120\.0|7200\.0|0\.15|0\.30|240\.0)|:=\s*(120\.0|7200\.0|0\.15|0\.30|240\.0)\b' \
	"$NUM_WL"

# ④ proto/venue 枚举再造启发式（proto='x' 字面谓词出名册读面宿主外；
#   同线含 fin_venue_roster=名册读面本体非枚举再造——多行 SQL 续行
#   形由文件白名单兜底）
PROTO_WL='internal/mt5/|internal/portfolio/hunger/depenum_mt5venue\.go|internal/data/okxpriv/'
scan "proto 字面谓词（名册读面宿主外枚举再造嫌疑）" \
	"proto\\s*=\\s*'[a-z_]+'|proto\\s*==\\s*\"[a-z_]+\"" \
	"$PROTO_WL" 'fin_venue_roster|//.*proto\s*='

if [ "$FAIL" -eq 0 ]; then
	echo "✅ [hardcode] 四面断言净（venue 名册真源/缺省单源/-SWAP 谓词收编/proto 枚举不再造）"
fi
exit "$FAIL"
