-- fin_0251 教训矿消费环（工单O-38942——verdicted 验尸案死档接消费）：
--   ① v_lesson_family_stats 物化视图：(template_ref,reason)→deaths/
--     net_usd/avg_age_days/verdicted_count——族级尸账供人工审计直读
--     +regime 喊人带尸检上下文（B 面；A 面签名粒度判据走基表直查——
--     视图粒度无 inst/direction 不可替）。refresh=生命周期官 scan 拍
--     节流 1h（lessongate.go lessonStatsRefresh——非 cron 依赖）。
--   ② app_config lifecycle.lesson_dead_sig=true——尸签抑制闸开关
--     （默认开；置 false=全闸停摆回滚不撤码；lifecycle 域=DM 写域册
--     在册 depenum 无 orphan）。
--   ③ 读面补授：o_fin_ro/roster_rw SELECT（审计直读面——fin_0249
--     同款判例）。
--   ④ fin_proposal 退役死因词表扩 'dead_signature'（fin_0085 封闭词表
--     补值——spec 钦定标死签名；md_1108 对偶两链归一化全等，projector
--     镜像写 retire_reason 不撞 CHECK）。

CREATE MATERIALIZED VIEW IF NOT EXISTS v_lesson_family_stats AS
SELECT numbers->>'template_ref' AS template_ref,
       numbers->>'reason'        AS reason,
       count(*)                 AS deaths,
       round(sum(COALESCE((numbers->>'realized_usd')::float8,0)
                + COALESCE((numbers->>'paper_net_usd')::float8,0))::numeric,2) AS net_usd,
       round(avg((numbers->>'age_days')::float8)::numeric,2) AS avg_age_days,
       count(*) FILTER (WHERE status='verdicted') AS verdicted_count
FROM fin_econ_lesson
WHERE kind='retire_postmortem'
GROUP BY 1, 2;

INSERT INTO app_config(key,value,updated_by) VALUES
  ('lifecycle.lesson_dead_sig', 'true'::jsonb, 'fin_0251')
ON CONFLICT (key) DO NOTHING;

ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_reason_vocab;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_reason_vocab
	CHECK (retire_reason IS NULL OR retire_reason IN (
		'eval_rejected','stale_expired','armed_ttl_expired','falsified_回炉',
		'superseded_armed','venue_untradeable','manual','other','legacy_unrecorded',
		'dead_signature')) NOT VALID;
ALTER TABLE fin_proposal VALIDATE CONSTRAINT fin_proposal_reason_vocab;

DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON v_lesson_family_stats TO o_fin_ro';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT ON v_lesson_family_stats TO roster_rw';
	END IF;
END $$;
