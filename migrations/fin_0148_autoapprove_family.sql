-- 0148 GW-001 段C2/C3 机批册按族入册 + 草案并票态。
-- ①meta.autoapprove.registry 按族入册（GW-001 段C2）：键=签名族字面——
--   rootcause:<family>（族谱=internal/ledger/diagnosis.go diagFamilies.ID，
--   族↔册机械对账桩在 diagnosis_test——新族入库必同拍补册否则测试红）
--   +postmortem:*（段D 裁定席 bug 案升件签名族通配）。
--   值=该族草案的安全钦定面（诊断/复盘类草案的工作面=经验库条目录——
--   记 lesson/复盘件非修码件）；修法面落钱径的族永不机批——负目录
--   （scripts/meta/autoapprove_negative_catalog.tsv）由 dm_autoapprove
--   落码全册扫描 fail-closed（非散文断言，GW-001 段C2 钦定）。
-- ②fin_ticket_draft status 加 'merged'（段C3 同签名族 24h 并票——后到
--   草案并入先到票，ticket 列载吸收票号；原 CHECK 三态扩四态）。
-- 幂等：jsonb || 浅合并（新键追加旧键不动）；约束重放 IF EXISTS。
UPDATE app_config SET value = value || '{
  "rootcause:unit_mismatch": ["docs/工程经验库/条目", "docs/经济经验库"],
  "postmortem:*":            ["docs/工程经验库/条目", "docs/经济经验库"]
}'::jsonb
WHERE key='meta.autoapprove.registry';

INSERT INTO app_config(key,value) VALUES ('meta.autoapprove.registry','{
  "观测盲:*":             ["internal/research/sigdist", "internal/portfolio/hunger"],
  "写败:LIFECYCLE-HUNGER":["internal/portfolio/hunger"],
  "写败:LIFECYCLE-MINER": ["internal/portfolio/hunger"],
  "账链断:ops_event":     ["internal/platform/opschain"],
  "rootcause:unit_mismatch": ["docs/工程经验库/条目", "docs/经济经验库"],
  "postmortem:*":            ["docs/工程经验库/条目", "docs/经济经验库"]
}'::jsonb)
ON CONFLICT (key) DO NOTHING; -- 键缺兜底（fin_0143 种子被清的极端面；键在=上行 UPDATE 已并）

ALTER TABLE fin_ticket_draft DROP CONSTRAINT IF EXISTS fin_ticket_draft_status_check;
ALTER TABLE fin_ticket_draft ADD CONSTRAINT fin_ticket_draft_status_check
  CHECK (status IN ('pending','materialized','dismissed','merged'));
