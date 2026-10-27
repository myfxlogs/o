-- fin_0079 死尾行清面（T-0197④——md_ticks 离集陈键+md_mark_index
-- mark_price/spot_last 退役频道僵行；有界表自净语义钦定）。
-- 一次性清面：退役频道行+离集陈键删（谓词读败 fail-safe 不删——
-- 本件只删退役频道死件，键级谓词面走清道夫日拍规则域）。
DELETE FROM md_mark_index WHERE channel IN ('mark_price','spot_last'); -- T-0197④返修：退役频道零写者=全行僵尸（谓词不挂——活跃标的僵行同清）
DELETE FROM md_ticks WHERE inst_id NOT IN (SELECT inst_id FROM fin_active_set)
  AND ts < now()-interval '7 days';
