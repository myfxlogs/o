# D-0024 入金双缺口——deposit 自动检测入账 + funding 户闲资停泊扫（D-286 钱不眠）

## 态：待施工
## 原编号：T-0149（2026-10-26 前后台分系改编——后台 D 系/前端 T 系）
## 域：ledger 记账 / earn 停泊径
## 源出：2026-10-26 实案——业主 OKX 入金 2000 USDT（depId 440867191，
  USDT-Omni state=2 实拉在案）触发账实差 $1999.57 告警；决策席手工三件清：
  fin_venue_bill 源行→deposit cashflow→SavingsPurchase 手停 Earn+留痕

## 病灶双件
1. **入金检测缺席**：venue 入金无自动入账径——deposit-history
   （/api/v5/asset/deposit-history）读面不在任何周期/事件径；业主注资=
   账本外增量，对账如期告警（告警正确，无检测=缺陷）。首笔入金靠
   opening_owner_declared 手工行，次笔靠决策席手清——**第三次起不该再手**。
2. **funding 户闲资不扫**：earn depositIdle 闲资检测只读交易户 avail
   （okxAvailFromEvents）——入金天然落 funding 户（OKX 入金默认归口），
   钱到即闲而扫面看不到（本轮 2000 若不手动，0% 息躺到永远）。D-286
   钱不眠钦定：闲资停泊=系统分内，此处是机器面真缺口非业主配合面。

## 施工
- deposit 检测径：deposit-history 事件化/低周期补漏拉取→fin_venue_bill
  （inst_type='ASSET' type='deposit' bill_id=dep-<depId> 幂等）→
  fin_cashflow kind='deposit'（occurred_at=venue ts）。无推送面 venue=REST
  补漏正当（D-0008：无流面 venue 轮询是补漏非主径）。
- funding 户闲资扫入 depositIdle：闲资额=交易户 avail + funding 户
  USDT avail（REST /asset/balances——低频）；funding 段直 SavingsPurchase
   免摆渡（funding→Earn 单跳），交易户段维持现三跳。阈=现 sweep_min 同键。
- 桩锚：seed deposit-history→bill+cashflow 幂等双行；funding 有余额→
  直购零摆渡；双源闲资合并判阈。

## 验收锚
- 再入金：零告警零手清（对账自闭）+ funding 户闲资下一拍自动停泊。
- 现网残留对账差归零（本轮手工行已收——复核权在桩）。

## 附
- 手工处置档（决策席代机件）：fin_venue_bill id=699780（dep-440867191
  raw venue JSON）、fin_cashflow id=212（deposit 2000）、
  fin_risk_decision id=32942（earn_deposit 2000）、earn_parking 快照行。
