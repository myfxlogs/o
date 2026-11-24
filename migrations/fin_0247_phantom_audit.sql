-- 0247 earn-redeem-phantom-audit（工单O-36422——业主裁示「对账面按币量
-- 对拍，USD 折算只归 NAV 估值层禁入 custody 差」裁执+BTC 注资映射根修）：
--
-- ①豁免行 #18(dcd_mtm_drift:11448262,$53)作废——resolver 语义=BTC 名义
--   量差×mark 现价折 USD=对账层跑 USD 折算差自豁免（owner 裁示禁面）；
--   且三分量对拍实证 venue notionalSz=0.0207000000000000 冻结=基线凭据
--   0.0207 BTC，币量差恒 0——resolver 自部署起结构性盲算 $0（|0-53|<2×53
--   破带带内静默采用，锚值系差额凑数非因果分解）。作废走 expires 语义+
--   reason 追加（fin_0244 #20-22 同式）；resolver 注册表键同步退役。
--   earn_accrual_gap(#19)USDT 本币应计=币量面内合法成分不裁撤；
--   静态行 #23/#17 照旧。
--
-- ②BTC 注资簿记补腿（depId=439722036 venue 实证在案）：
--   业主 2026-10-21 链上充 0.02 BTC（BTC-Bitcoin, state=2）——DepositScanner
--   T-0149 非 USDT 闸跳过不留 bill，金源对簿记全盲；后申购 DCD
--   11448262(0.0207BTC 名义,基线凭据 id=165 手工件早于 O-33362 自动化)
--   缺 dcd_out counted 腿。「BTC 等账外币单不补」裁定的前提=金源账外
--   ——deposit 入账后前提消解，补全链：bill 源行+deposit 现金腿+
--   dcd_out 申购腿（-0.0207BTC,source_ref=165 同 dcdBaselineWrite 机械化
--   形——0.02 新注资+0.0007 存量入产品，存量段无对应入账=诚实残差面）。
--   幂等：bill_id ON CONFLICT + cashflow NOT EXISTS 双闸；②三条 INSERT
--   全部挂事件指纹闸（ops_event#33341 手工簿记误记行实存=本案环境
--   判据）——fin_0043 在全迁移域种子同 ordId 凭据，凭据存在性不能
--   区分生产/测试夹具；指纹闸使测试库零写（迁移不污染夹具域），
--   生产误记行在案即补账，行被清理后重放亦零写=自域自销。

UPDATE fin_known_residual
	SET expires_at=now(),
	    reason=reason||'｜作废判词:O-36422口径界裁——custody纯币量对拍,USD折算漂移退NAV层禁入对账面;resolver键dcd_mtm_drift同步退役(venue名义量冻结0.0207=基线,币量差恒0结构性盲算);币量面BTC键径独立对拍在案'
	WHERE id=18;

-- ②a bill 源行（dep-<depId> 幂等键——scanner 复扫撞冲突自然跳过）。
INSERT INTO fin_venue_bill
	(bill_id,ts,inst_id,inst_type,ccy,type,sub_type,bal_chg,fee,pnl,sz,px,raw)
SELECT 'dep-439722036','2026-10-21 05:24:20+00','','ASSET','BTC','deposit','',0.02,0,0,0,0,
	'{"depId":"439722036","ccy":"BTC","amt":"0.02","chain":"BTC-Bitcoin","state":"2","ts":"1792560260000","backfill":"O-36422 venue deposit-history 实证行"}'::jsonb
WHERE EXISTS (SELECT 1 FROM fin_cashflow
		WHERE source_table='ops_event' AND source_ref=33341
		  AND kind='deposit' AND ccy='USDT' AND amount=1655.94)
  AND NOT EXISTS (SELECT 1 FROM fin_venue_bill WHERE bill_id='dep-439722036');

-- ②b deposit 现金腿 +0.02 BTC（source_ref=fin_venue_bill.id 引用链——
--   bill 行缺席即零写，②a 闸自然级联）。
INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
SELECT 'deposit','okx','BTC',0.02,b.id,'fin_venue_bill','2026-10-21 05:24:20+00'
FROM fin_venue_bill b WHERE b.bill_id='dep-439722036'
AND NOT EXISTS (SELECT 1 FROM fin_cashflow f
	WHERE f.kind='deposit' AND f.source_table='fin_venue_bill' AND f.source_ref=b.id);

-- ②c dcd_out 申购腿 -0.0207 BTC（DCD-11448262 名义全额——source_ref=基线
--   凭据 id=165,occurred_at=venue cTime 2026-10-21 12:21:46 同
--   dcdBaselineWrite/dcdOccurTS 机械化形；凭据行内容双校防 id 巧合）。
INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
SELECT 'dcd_out','okx','BTC',-0.0207,165,'fin_risk_decision','2026-10-21 12:21:46+00'
WHERE EXISTS (SELECT 1 FROM fin_cashflow
		WHERE source_table='ops_event' AND source_ref=33341
		  AND kind='deposit' AND ccy='USDT' AND amount=1655.94)
  AND EXISTS (SELECT 1 FROM fin_risk_decision
		WHERE id=165 AND intent->>'op'='dcd_baseline'
		  AND intent->>'ordId'='11448262' AND intent->>'venue'='okx')
  AND NOT EXISTS (SELECT 1 FROM fin_cashflow f
	WHERE f.kind='dcd_out' AND f.ccy='BTC' AND f.source_ref=165);
