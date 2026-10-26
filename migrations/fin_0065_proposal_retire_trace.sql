-- T-0167 分链：桥表 fin_proposal 退位溯源列+活行索引，双链同形（md 链 md_0065 同文）。
-- 0065 T-0161 eval↔proposal 联动清扫+陈旧提案回收——fin_proposal 退位溯源列。
-- 病灶：提案退役（T-0069 淤积闸/T-0136 超龄/本票 eval_rejected 联动与
-- stale_expired 陈旧回收）至今零落痕——635k fin_retired 里查不出任何一行
-- 为何退。补 retire_reason（闭集由写方语义定：eval_rejected/stale_expired/
-- armed_ttl_expired；存量行 NULL=机制前溯不虚构）+retired_at（退位时点）。
ALTER TABLE fin_proposal
    ADD COLUMN IF NOT EXISTS retire_reason text,
    ADD COLUMN IF NOT EXISTS retired_at timestamptz;

-- 卫生闸每拍索引面（lifecycle 逐拍跑联动/陈旧谓词——全表 63.5万 vs 活行数百，
-- 部分索引只盖活行使每拍扫描 O(活行)，不建=每拍 seq scan 洪泛）。
CREATE INDEX IF NOT EXISTS fin_proposal_live_eval ON fin_proposal (eval_id)
    WHERE state IN ('fin_proposed', 'fin_armed');
CREATE INDEX IF NOT EXISTS fin_proposal_proposed_age ON fin_proposal (created_at)
    WHERE state = 'fin_proposed';
