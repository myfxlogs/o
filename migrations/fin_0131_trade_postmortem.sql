-- 0131 T-0420 平仓监察件——结案判定机+裁定席监察案源。
-- 病灶（业主判词）：「开仓/平仓了没人管——赚了还好，亏了后如何处理没人管」。
--   ops_decision 已记开平理由（T-0415 记忆面），读理由复盘的机器不存在——
--   记录只进不出。本迁=监察面三配：
--   ① ops_decision fin 域 action 闭集扩 'outcome'——结案判词行（正常结案
--     verdict='expected' 落此——记忆×结果对照案闭环；seat 裁词集不扩）。
--   ② fin_econ_lesson kind 闭集扩 'trade_postmortem'——异常结案监察案入
--     裁定席 pending 集（UNIQUE(kind,entity_ref) 天然幂等，一仓一案不重行）。
--   ③ guard.postmortem.* 判据键三件种子——时长阈/结算幅阈/触发认知集
--     参数化（guard.* 守仓参数 dm 写域分册在案，热启键缺=判据链 fail-closed）。
-- 约束换名纪律：原 CHECK 为内联无名式（自动名随落库序漂移）——按约束定义
--   子串定位真名 DROP 再 ADD 具名约束（部署窗幂等：定位不到=已新集，ADD
--   具名重放撞名即错显形非静默）。

-- ① fin 域 action 集扩 outcome（定位旧域/action 组合 CHECK 真名——def 含
--   'open' 字面量者即本约束；btrim(reason) 另约束不动）。
DO $$
DECLARE c text;
BEGIN
	SELECT conname INTO c FROM pg_constraint
		WHERE conrelid='ops_decision'::regclass AND contype='c'
			AND pg_get_constraintdef(oid) LIKE '%''open''%';
	IF c IS NOT NULL THEN
		EXECUTE format('ALTER TABLE ops_decision DROP CONSTRAINT %I', c);
	END IF;
END $$;
DO $$ BEGIN
	ALTER TABLE ops_decision ADD CONSTRAINT ops_decision_action_chk
		CHECK ((domain='fin'  AND action IN ('open','close','adjust','reject','outcome'))
		    OR (domain='seat' AND action IN ('裁决','驳回','参数调整','升件裁量')));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
COMMENT ON COLUMN ops_decision.action IS 'fin=open|close|adjust|reject|outcome（T-0420 结案判词行）；seat=裁决|驳回|参数调整|升件裁量';

-- ② econ kind 集扩 trade_postmortem（同式定位旧 kind CHECK）。
DO $$
DECLARE c text;
BEGIN
	SELECT conname INTO c FROM pg_constraint
		WHERE conrelid='fin_econ_lesson'::regclass AND contype='c'
			AND pg_get_constraintdef(oid) LIKE '%falsified_verdict%';
	IF c IS NOT NULL THEN
		EXECUTE format('ALTER TABLE fin_econ_lesson DROP CONSTRAINT %I', c);
	END IF;
END $$;
DO $$ BEGIN
	ALTER TABLE fin_econ_lesson ADD CONSTRAINT fin_econ_lesson_kind_chk
		CHECK (kind IN ('falsified_verdict','settled_loss','cost_diverge',
			'venue_degrade','flip_exit','semantic_caliber','trade_postmortem'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
COMMENT ON COLUMN fin_econ_lesson.kind IS '案类闭集——trade_postmortem=平仓监察案（T-0420：结案判定机异常件入裁，entity_ref=position#<id>）';

-- ③ 结案判据键三件（缺任一键=判据链 fail-closed 跳判+计数，与 guard 同律）。
INSERT INTO app_config(key,value) VALUES
  ('guard.postmortem.min_hold_minutes','5'),
  ('guard.postmortem.settled_anomaly_pct','20'),
  ('guard.postmortem.known_triggers','["funding_flip","funding_flip_preempt","delta_drift","max_hold","strategy_degraded","strategy_retired","dd_30d","manual"]')
ON CONFLICT (key) DO NOTHING;
