-- 0014 paper 同标的去重：direction 入策略行+active 部分唯一索引
-- （同 venue+inst_id+direction 同时最多一个非 retired 策略行——armed 复用不叠仓）。
-- dedupe 前步（migration-unique-index-dirty-data 坑判据：唯一索引迁移须自带
-- 前步——同键旧行只留最新，余置 retired 再建索引；幂等可重放）。
ALTER TABLE fin_strategy ADD COLUMN IF NOT EXISTS direction text NOT NULL DEFAULT 'short_perp_long_spot';
UPDATE fin_strategy s SET state='retired'
WHERE ctid IN (
  SELECT ctid FROM (
    SELECT ctid, ROW_NUMBER() OVER (
      PARTITION BY venue, inst_id, direction
      ORDER BY id DESC) rn FROM fin_strategy WHERE state<>'retired') q
  WHERE rn>1);
CREATE UNIQUE INDEX IF NOT EXISTS fin_strategy_active_dedupe
    ON fin_strategy(venue, inst_id, direction) WHERE state <> 'retired';
