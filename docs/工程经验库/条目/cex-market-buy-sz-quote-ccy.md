---
schema_version: 2
id: cex-market-buy-sz-quote-ccy
kind: pitfall
risk: medium
status: verified
keywords: [市价买单, sz, tgtCcy, quote_ccy, base_ccy, USDT计价]
scope: [cex 现货市价买, okx, 下单量口径]
summary: CEX 市价买单 sz 默认=计价币（USDT 金额）口径非基准币——要按基准币数量下单必须显式 tgtCcy=base_ccy
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-api-spec, okx-exec]
---
# cex-market-buy-sz-quote-ccy —— CEX 市价买单数量默认=计价币（USDT）口径非基准币

## 坑
OKX spot **市价买单**的 `sz` 默认按**计价币（USDT 金额）**解释，不是基准币数量——传 BTC 数量 0.00030914 被当 $0.0003 买单拒单。业主钦定口径：**CEX 所有计价都是 USDT**——市价买单的金额语义先想 USDT，要按基准币数量下单必须显式 `tgtCcy=base_ccy`。

## 实证
PHASE exec_id=3（2026-10-24 真户）：leg1 BTC-USDT-SWAP sell 0.03 filled；leg2 BTC-USDT spot buy sz=0.00030914 failed（本意 $25 等值 BTC，被解释为 $0.0003 USDT 买单）。leg1 已按平仓语义真回滚零残仓。拒码因走 envelope APIError 径未留痕（T-0042 同修）。

## 反证条件
- 任何 venue 的市价买单面：下单前必核 sz 计量币种（quote 默认/base 显式）——sell 市价单默认 base、buy 市价单默认 quote 是 OKX 方言，**禁跨方向/跨 venue 推广假设**。
- 换用 limit 单/IOC 或 tgtCcy 显式声明可绕——但口径必须在代码注释钉死。
- 同族病灶：ctVal 币本位（T-0037）、posSide 户型（T-0040）——**venue 数量/金额语义永不猜，下单时点读 instruments 元数据+方言卡**。

## 官方口径（docs-v5 下单端点核证 2026-10-24）
- **币币/币币杠杆市价单**：sz 单位由 `tgtCcy` 决定——买单默认 quote_ccy（USDT 金额）、卖单默认 base_ccy；显式 `tgtCcy=base_ccy` 则按交易货币数量
- **合约单（交割/永续/期权）**：sz=张数——面值 = `sz×ctVal×markPx`（**正向=USDT 本位**）或 `sz×ctVal`（**反向=币本位 USD 计价**）；ctVal/ctType 取 instruments
- **开平仓模式（long_short）**：平仓单天然只减仓逻辑（reduceOnly 传不传都生效），sz 超持仓=整单拒
- 文档源：okx.com/docs-v5/zh 下单端点 sz/tgtCcy 参数表（trick 页无此细目）

## 机械化
- T-0042：OrderParams.TgtCcy + spot 市价买单传 base_ccy + 测试锚。
- skills/okx-exec 补 tgtCcy 方言条目（T-0042 同票落）。
