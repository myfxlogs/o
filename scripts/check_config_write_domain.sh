#!/usr/bin/env bash
# check_config_write_domain.sh —— app_config 写域二分册机检闸（T-0376 锚④
# 静态断言）。分册正本=scripts/config-write-domain.tsv；Go 径=internal/
# platform/config/dmwrite.go（dmWritableGlobs/ownerKeyGlobs 同集镜像）。
# 五断言：
#   ① 集同：TSV dm/owner 行集 == dmwrite.go 两 glob 集逐字（漂移即红）；
#   ② 硬界：dm 行零凭证形、零业主面硬键（资金上限/停机总闸/对账容差
#      ——碰即红）；TSV 两集交叠=红（owner 行优先语义不允许交叠歧义）；
#   ③ 覆盖棘轮：Go 消费宇宙（config 读器调用点+*Key 常量定义的键字面量）
#      每键必被「凭证形 regex ∨ 分册行」覆盖——未登记=红（新键须入册，
#      fail-closed 归 owner 是运行时闸，本断言逼文档面同步）。
#   ④ 行为级对拍=go test（TestClassifyKeyDomain/TestDMSetHardBoundsUnreachable
#      ——check_all 全谱跑；集同①保证 TSV 行为=Go 行为的文档镜像不失真）。
#   ⑤ 裸 Get 白名单：internal/platform/config 包外非测试 Go 件对字面量键
#      的裸 .Get 读点「文件:键」必须在白名单内——标量键禁裸 Get（走
#      GetBool/GetFloat/GetInt/GetString 宽容读器；canon
#      config-json-type-fragility 复发实证=killSwitchOn 单型解 jsonb 布尔
#      致回撤硬线失明三周，工单#20799），清单/映射键现存读点全列白名单。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"

TSV=scripts/config-write-domain.tsv
GOFILE=internal/platform/config/dmwrite.go
fail() { echo "❌ [config-write-domain] $1" >&2; exit 1; }
[ -f "$TSV" ] || fail "分册缺失: $TSV"
[ -f "$GOFILE" ] || fail "写径件缺失: $GOFILE"

# —— ① 集同（TSV 行集 vs Go glob 集）——
tsv_dm=$(awk -F'\t' '$2=="dm" && $1 !~ /^#/ {print $1}' "$TSV" | sort)
tsv_owner=$(awk -F'\t' '$2=="owner" && $1 !~ /^#/ {print $1}' "$TSV" | sort)
go_block() { # $1=var 名——取 var 块内引号字面量（一行一 glob）
  sed -n "/var $1 = \[\]string{/,/^}/p" "$GOFILE" | grep -oE '"[^"]+"'
}
go_dm=$(go_block dmWritableGlobs | tr -d '"' | sort)
go_owner=$(go_block ownerKeyGlobs | tr -d '"' | sort)
[ -n "$go_dm" ] || fail "dmWritableGlobs 块解析空（镜像面结构漂移）"
[ -n "$go_owner" ] || fail "ownerKeyGlobs 块解析空"
diff <(echo "$tsv_dm") <(echo "$go_dm") >/dev/null \
  || fail "dm 行集漂移——TSV↔dmWritableGlobs 不同（diff: $(diff <(echo "$tsv_dm") <(echo "$go_dm") | head -5 | tr '\n' ' '))"
diff <(echo "$tsv_owner") <(echo "$go_owner") >/dev/null \
  || fail "owner 行集漂移——TSV↔ownerKeyGlobs 不同（diff: $(diff <(echo "$tsv_owner") <(echo "$go_owner") | head -5 | tr '\n' ' '))"

# —— ② 硬界（dm 行零凭证形/零业主面硬键；两集零交叠）——
CRED='api_?key|secret|token|passphrase|password|account_id|private_key'
HARDKEYS='risk\.max_notional_usd|risk\.kill_switch|ledger\.recon_tol_usd|trx\.stake_address|trx\.rental_payers'
cred_hit=$(echo "$go_dm" | grep -iE 'api_?key|secret|token|passphrase|password|account_id|private_key' | head -3)
[ -z "$cred_hit" ] || fail "dm 集含凭证形键: $cred_hit（锚④硬界）"
hard_hit=$(echo "$go_dm" | grep -E "$HARDKEYS" | head -3)
[ -z "$hard_hit" ] || fail "dm 集含业主面硬键: $hard_hit（锚④硬界）"
overlap=$(comm -12 <(echo "$go_dm") <(echo "$go_owner") | head -3)
[ -z "$overlap" ] || fail "dm/owner 两集交叠: $overlap（判定歧义禁）"
# Go 判定序=凭证 regex 先于 dm 行——dm 行即含凭证词也被 regex 压制，但
# 分册行面歧义本身=文档病，①②同钉。

# —— ③ 覆盖棘轮（Go 消费宇宙逐键登记）——
UNIVERSE=$(
  { grep -rhoE '\.(Get|GetString|GetBool|GetFloat|GetInt|GetStringList)\("[a-z][a-z_0-9]*(\.[a-z_0-9]+)+"' \
      internal cmd tools --include='*.go' --exclude='*_test.go' 2>/dev/null | grep -oE '"[^"]+"'
    grep -rhoE '(Key|key)[[:space:]]*=[[:space:]]*"[a-z][a-z_0-9]*(\.[a-z_0-9]+)+"' \
      internal cmd tools --include='*.go' --exclude='*_test.go' 2>/dev/null | grep -oE '"[^"]+"'
  } | tr -d '"' | sort -u
)
[ -n "$UNIVERSE" ] || fail "键宇宙扫描空（消费面结构漂移——读器/常量口径核对）"
unreg=$(echo "$UNIVERSE" | while IFS= read -r key; do
  echo "$key" | grep -qiE "$CRED" && continue # 凭证形=regex 硬拒域免册
  matched=0
  while IFS=$'\t' read -r pat cls _; do
    [ "${pat:0:1}" = "#" ] && continue
    case "$key" in
      $pat) matched=1; break ;;
    esac
  done < "$TSV"
  [ "$matched" = 0 ] && echo "$key"
done | head -8)
[ -z "$unreg" ] || fail "消费宇宙未登记键（入册后 DM 才可写——fail-closed 域）: $(echo "$unreg" | tr '\n' ' ')"

# —— ④ 行为级对拍=go test（见头注——本脚本零探针）——

# —— ⑤ 裸 Get 白名单闸（标量键禁裸 .Get——config 包外字面量键裸读现存点全列；
#    「文件:键」粒度，新增读点未登记即红）——
BARE_GET_ALLOW=$'internal/mt5/bridge.go:bridge.mt5.symbols
internal/mt5/collector.go:mt5.gateway_addr
internal/mt5/collector.go:mt5.symbols
internal/mt5/session.go:mt5.gateway_addr
internal/mt5/session.go:mt5.gateway_fallbacks
internal/venue/onboard/mapgate.go:okx.symbols
internal/portfolio/hunger/coverage.go:okx.symbols
internal/repair/watch.go:repair.feeds
internal/repair/watch.go:repair.guard
internal/research/sigdist/xvenue.go:bridge.xvenue.map
internal/research/sigdist/xshadow.go:okx.symbols
internal/data/okxws/depth5.go:bridge.depth5.symbols
cmd/core/dialect_submit.go:okx.symbols'
bare=$(grep -rnE '\.Get\("[a-z0-9_]+\.[a-z0-9_.]+"' internal cmd tools \
    --include='*.go' --exclude='*_test.go' 2>/dev/null \
  | grep -v '^internal/platform/config/' \
  | sed -E 's/^([^:]+):[0-9]+:.*\.Get\("([a-z0-9_.]+)"\).*/\1:\2/' | sort -u)
bad=$(comm -23 <(echo "$bare") <(echo "$BARE_GET_ALLOW" | sort -u))
[ -z "$bad" ] || fail "裸 Get 白名单外读点（标量键禁裸 .Get——走 GetBool/GetFloat/GetInt/GetString；清单/映射键读点须先入白名单）: $(echo "$bad" | tr '\n' ' ')"

echo "✅ [config-write-domain] 集同+硬界+覆盖+裸Get四断言全绿（dm=$(echo "$go_dm" | wc -l) 行 owner=$(echo "$go_owner" | wc -l) 行 universe=$(echo "$UNIVERSE" | wc -l) 键 bare=$(echo "$bare" | wc -l) 读点）"
