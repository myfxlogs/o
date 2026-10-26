# T-0149 入金双缺口——deposit 自动检测入账 + funding 户闲资停泊扫（D-286 钱不眠）

## 态：✅ 验收毕投产（241f79a 合入·决策席 2026-10-26）
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

## 结果回填

- ①deposit 检测径：internal/ledger/funding/deposit.go——deposit-history
  state=2 完成行→fin_venue_bill（bill_id=dep-<depId> 幂等键）→
  fin_cashflow kind='deposit'（source_ref 引用链禁推断记账）；同挂
  billsHook 事件帧同拍扫（无推送面=REST 补漏正当面）。
- ②funding 户闲资扫入 sweepIdle：AttachIdleFunding（/asset/balances
  USDT availBal 现读件）——funding 段超阈直 SavingsPurchase 单跳零
  摆渡；交易户段三跳零动；58350/歧态同规 fail-closed。
- 桩锚双绿：TestDepositScanIdempotent（重复巡检不重账+非完成态不入账）、
  TestFundingIdleDirectPurchase（funding 500 直购零摆渡）。

## 二返修（三锚）

- ①USDT 谓词闸：r.Ccy!="USDT"→rej 计数+skip 不入账不留 bill（D-0021
  TRX 自营边界——非 USDT 入金非基金注资入账=虚增 NAV）。
- ②逐行错误隔离：单行 begin/bill/cashflow/commit 败=log+rej+continue
  不 abort 整扫（毒行饿死后续行=现产实证病灶根修）。
- 桩锚：TestDepositScanUSDTGate（TRX/BTC 行零写+后续 USDT 照入账）
  +TestFundingIdleBackoff（连败两拍第二拍不重发）。

## 三返修（账基线截止谓词）

- Scan 入金行加基线截止谓词：occurred_at ≤ fin_cashflow 最早行 occurred_at
  （账基线锚）→pre_baseline 计数+skip 不入账不留 bill——bill_id 幂等只
  防重不防史病灶根修（venue 全史前身时代入金 $420,803 当注资入账=虚增
  8 倍实证）；postdate 基线行照常入账。
- 桩锚：TestDepositScanBaselineCutoff（pre1 零写无 bill+post1 入账）绿；
  前两桩测补 seedBaseline 基线行（基线参照不在即全 skip——按设计语义）。
