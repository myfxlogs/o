-- 0250 nft-baseline-double（工单O-37868——NFT 双腿重簿根修+迟簿竞争机制闸）：
--
-- 背景：native_baseline 凭据 #35360（2026-11-23 02:44 首武装——arm 时
-- fin_cashflow 尚无 NFT 行→resid=全额 51,788,586.877993 建）与迟簿
-- deposit 现金腿 #1808（bill 源 12701247，occurred 2025-08-25，
-- ts 2026-11-24 05:02 落账——O-36422 币形闸放开后 scanner 首扫回填
-- 史前入金批）同笔入金双计→账本净 103.58M=venue 51.79M 精 2 倍，
-- USD 残差随 NFT 价漂复火（豁免栈叠 #24/#26 治标在案）。
--
-- ①负腿凭据（append-only 冲销——B 案 UPDATE 删凭据=改写历史已否）：
--   op=native_baseline okx/NFT amt=-51,788,586.877993 → NativeBaselineAmt
--   Σ(venue,ccy)=0 归零；lateSrc='1808' 与迟簿补偿机制幂等键同域——
--   compensateLateBaselineBooks 见 #1808 已补不再写（手腿=机腿同物）。
--   写入闸=#35360 行内容双校（id+op+venue+ccy+amt 全等防 id 巧合；
--   测试库无此凭据=零写不污染夹具域，同 fin_0247 指纹闸律）。
--
-- ②豁免 #24/#26 作废（expires 语义+reason 追加，fin_0247#18 同式）——
--   币量差归零后 USD 残差豁免反转成负假差源；expires_at>now() 闸使
--   重放零写（幂等）。
--
-- ③机制闸判词=要闸（internal/ledger/native_base.go
--   compensateLateBaselineBooks，reconcile 拍内自愈）：armed 键上
--   occurred_at<arm_ts ∧ ts>arm_ts 的 custody 现金流逐行补
--   native_baseline 负腿（amt=-amount，lateSrc=cfid 幂等）——迟簿
--   竞争面结构性存在（scanner 史前回填/迁移补账/币形闸放开首扫皆可
--   复发；SHIB/BTT 靠 withdraw 对冲巧合幸免非设计），不修=每次迟簿
--   都靠人工迁移根修。

-- ①负腿冲销（双校闸+#1808 幂等锚——机制闸同键不双补）。
INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
SELECT jsonb_build_object('op','native_baseline','venue','okx','ccy','NFT',
		'amt',-51788586.877993::float8,'lateSrc','1808','corr','O-37868'),
	'allow','allow',1,'open'
WHERE EXISTS (SELECT 1 FROM fin_risk_decision
	WHERE id=35360 AND intent->>'op'='native_baseline'
	  AND intent->>'venue'='okx' AND intent->>'ccy'='NFT'
	  AND (intent->>'amt')::numeric=51788586.877993)
  AND NOT EXISTS (SELECT 1 FROM fin_risk_decision
	WHERE intent->>'op'='native_baseline' AND intent->>'venue'='okx'
	  AND intent->>'ccy'='NFT' AND intent->>'lateSrc'='1808');

-- ②豁免 #24/#26 作废（币量差归零后 USD 残差豁免=负假差源）。
UPDATE fin_known_residual
	SET expires_at=now(),
	    reason=reason||'｜作废判词:O-37868——NFT双簿根修:迟簿deposit #1808与基线#35360双计经负腿冲销Σ=0,币量差归零后USD残差豁免反转负假差;迟簿竞争面机制闸=compensateLateBaselineBooks在案'
	WHERE id IN (24,26) AND expires_at > now();
