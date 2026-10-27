#!/usr/bin/env bash
# test_tron_apis.sh —— TRON 双 API 源功能证明脚本（T-0192 业主令：API 搞清楚+
# 测试脚本先行，功能测通再动码）。只读端点零写零密钥值回显。
# 用法: test_tron_apis.sh <voter地址> [trongrid_key]
set -uo pipefail
VOTER="${1:?用法: test_tron_apis.sh <voter地址> [trongrid_key]}"
TGKEY="${2:-}"
APILIST="https://apilist.tronscan.org"
TRONGRID="https://api.trongrid.io"
probe() { # probe <标签> <curl参数...>
  local tag="$1"; shift
  local out code
  out=$(curl -sS --max-time 10 -w "\n%{http_code} %{time_total}s" "$@" 2>&1 | tail -2)
  code=$(echo "$out" | tail -1 | awk '{print $1}')
  local body=$(echo "$out" | head -1)
  printf "%-46s => %s  %s\n" "$tag" "${code:-ERR}" "$(echo "$body" | head -c 60)"
}
echo "== A. TronScan apilist（数据面：witness 年化/vote 分布）=="
probe "A1 witness 无key"                       "$APILIST/api/witness"
probe "A2 witness 带trongrid_key(跨服务对照)"   -H "TRON-PRO-API-KEY: $TGKEY" "$APILIST/api/witness"
probe "A3 vote?voter 无key"                    "$APILIST/api/vote?voter=$VOTER"
probe "A4 vote?voter 带trongrid_key(对照)"      -H "TRON-PRO-API-KEY: $TGKEY" "$APILIST/api/vote?voter=$VOTER"
echo "== B. TronGrid（节点面：listwitnesses/getaccount）=="
if [ -n "$TGKEY" ]; then
  probe "B1 listwitnesses 带key"               -H "TRON-PRO-API-KEY: $TGKEY" "$TRONGRID/wallet/listwitnesses"
  probe "B2 getaccount 带key(票权/votes)"       -H "TRON-PRO-API-KEY: $TGKEY" -X POST -H "Content-Type: application/json" -d "{\"address\":\"$VOTER\"}" "$TRONGRID/wallet/getaccount"
  probe "B3 getaccountreward 带key(待领取)"     -H "TRON-PRO-API-KEY: $TGKEY" -X POST -H "Content-Type: application/json" -d "{\"address\":\"$VOTER\"}" "$TRONGRID/wallet/getreward"
else
  echo "（无 trongrid key 参数——B 组跳过）"
fi
echo "== 结论判读 =="
echo "  A1 429/A3 429 = tronscan 无 key 不保 QPS（2025-08-31 政策）→ 需 tronscan key"
echo "  B1/B3 200 = trongrid 面 SG 可达 → witness/领奖读路可切 trongrid"
echo "  A3 vs B2/B3 = votes 双源对拍（地址同则票数应一致）"
