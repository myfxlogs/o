-- 0085 退役死因强制化（T-0236/D-0037 双优先①——fin_proposal 1487 无因退役
-- 复盘盲区根治）：封闭词表 CHECK+retired 必填 CHECK+存量证据回填。
-- 词表：eval_rejected/stale_expired/armed_ttl_expired/falsified_回炉/superseded_
-- armed/manual（人工令）/other（其他）/legacy_unrecorded（对不上证据统一桶）。
-- 顺序：先回填→后加约束（同迁移件原子——CHECK 落地即无裸因存量）。

-- 回填①eval 翻否链：关联 eval verdict=rejected 钉 eval_rejected
UPDATE fin_proposal p SET retire_reason='eval_rejected',retired_at=COALESCE(retired_at,now())
WHERE p.state='fin_retired' AND (p.retire_reason IS NULL OR p.retire_reason='')
AND EXISTS (SELECT 1 FROM fin_opp_eval e WHERE e.id=p.eval_id AND e.verdict='rejected');

-- 回填②超龄特征链：retired−created≥24h 钉 armed_ttl_expired（armed/proposed
-- 双陈旧径合并因——可回溯证据=龄超阈，细分径史无凭不猜）
UPDATE fin_proposal SET retire_reason='armed_ttl_expired'
WHERE state='fin_retired' AND (retire_reason IS NULL OR retire_reason='')
AND retired_at-created_at>=interval '24 hours';

-- 回填③兜底统一桶：对不上证据=legacy_unrecorded（不猜）
UPDATE fin_proposal SET retire_reason='legacy_unrecorded'
WHERE state='fin_retired' AND (retire_reason IS NULL OR retire_reason='');

-- 约束①词表封闭（NULL 放行非退役行——armed/proposed 态允许空因）
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_reason_vocab;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_reason_vocab
	CHECK (retire_reason IS NULL OR retire_reason IN (
		'eval_rejected','stale_expired','armed_ttl_expired','falsified_回炉',
		'superseded_armed','venue_untradeable','manual','other','legacy_unrecorded'));

-- 约束②retired 态必填（空=写败显形）
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_retire_reason_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_retire_reason_check
	CHECK (state<>'fin_retired' OR retire_reason IS NOT NULL);

-- 无因病源③（测试面实证）：fin_0061 untradeable 回扫触发器裸退提案——
-- 函数体重替补 venue_untradeable 因（0061 原语义零改，仅补溯源列）。
CREATE OR REPLACE FUNCTION fn_untradeable_retro() RETURNS trigger AS $$
BEGIN
  INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
  SELECT id, state, 'retired', 'venue_untradeable_retroactive:'||NEW.code
  FROM fin_strategy WHERE inst_id=NEW.inst_id AND state IN ('live','paper','proposed');
  UPDATE fin_strategy SET state='retired'
  WHERE inst_id=NEW.inst_id AND state IN ('live','paper','proposed');
  UPDATE fin_proposal SET state='fin_retired',
    retire_reason='venue_untradeable',retired_at=now()
  WHERE inst_id=NEW.inst_id AND state='fin_armed';
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
