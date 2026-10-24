#!/usr/bin/env bash
# 基线校准一次性行（T-0055——历史缺口书面归因后真值校准）：
#   ledger_net = Σcashflow(venue,ccy)；venue_eq = 现读聚合（trading eq+funding+savings amt）
#   差额落 recon_adjust 行（source 约定 recon_baseline_adjust）。
# 前置：0030 迁移已部署（kind 扩枚）；利息首拍播种已发生（bootstrap 事件行在案）。
# 用法：DSN=… calibrate_recon.sh okx USDT  （干跑看数：CAL_DRY=1）
set -euo pipefail
DSN="${DSN:?需要 DSN}"
V="${1:-okx}"; C="${2:-USDT}"
LED=$(psql "$DSN" -Atc "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE venue='$V' AND ccy='$C';")
# venue 现读三件（人工跑时点——生产对账件同名口径）
echo "ledger=$LED（venue 侧请以对账件现读为准，差=venue-ledger 落 adjust 行）" >&2
read -r -p "venue 现读净值: " VEN
ADJ=$(python3 -c "print(f'{float('$VEN')-float('$LED'):.4f}')")
echo "adjust=$ADJ ($V/$C)"
if [ -z "${CAL_DRY:-}" ]; then
  psql "$DSN" -c "INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
    VALUES('recon_adjust','$V','$C',$ADJ,0,'recon_baseline_adjust',now());"
  echo "校准行已落——recon 静默成立判据=|差|<容差"
fi
