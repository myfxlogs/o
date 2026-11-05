-- 0102 T-0314 经济经验库自闭环锚表（事件锚定+digest+双消费腿）——
-- fin_econ_lesson 评史官唯一写者（fin 池写 SG 面正本）；kind CHECK 集含
-- semantic_caliber（语义案也是案——口径边界/Earn 冒充两案入库回填）；
-- UNIQUE(kind,entity_ref) 幂等同案同窗不重行；md_venue_req_push 同款审计标。
CREATE TABLE IF NOT EXISTS fin_econ_lesson (
    id          bigserial PRIMARY KEY,
    ts          timestamptz NOT NULL DEFAULT now(),
    kind        text NOT NULL CHECK (kind IN
        ('falsified_verdict','settled_loss','cost_diverge','venue_degrade','flip_exit','semantic_caliber')),
    entity_ref  text NOT NULL,           -- venue/inst/position_ref 复合引用串
    numbers     jsonb NOT NULL DEFAULT '{}', -- EconLesson protojson（UseProtoNames）
    status      text NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','verdicted','canon')),
    verdict_ref text NOT NULL DEFAULT '',
    UNIQUE (kind, entity_ref)
);
COMMENT ON TABLE fin_econ_lesson IS
  '经济经验库锚表（T-0314）——事件锚定案+裁定回改点；digest O 群推读面';
INSERT INTO fin_data_retention(table_name,class,note) VALUES
  ('fin_econ_lesson','state','经济案锚表不删行（T-0314）')
ON CONFLICT (table_name) DO NOTHING;
-- 信息面非钱径通知开关：econ_digest 仅发经验案摘要，不代表资金事件。
INSERT INTO app_config(key,value) VALUES
  ('notify.feishu.econ_digest.enabled','true'::jsonb),
  -- R1：gatefx Apply 「不在插行」防线配键先 seed（两拍分权——seed 立册
  -- 裁定拍只 UPDATE 不创键）。
  ('gate.kill.funding_carry','false'::jsonb),
  ('gate.lesson.cost_diverge_pct','20'::jsonb)
ON CONFLICT (key) DO NOTHING;
-- 种案回填（钦定两行 pending——口径边界+Earn 冒充案入库）。
INSERT INTO fin_econ_lesson(kind,entity_ref,numbers) VALUES
  ('semantic_caliber','cmc-trx-vs-company','{"summary":"TRX/CMC 个人盘收入误并公司盈利——口径边界案（2026-11-03 业主两令）","retro":"docs/运维/周报口径.md 域界排除节 5d89c29c"}'),
  ('semantic_caliber','earn-interest-as-strategy-profit','{"summary":"Earn 停泊息冒充策略利润——周净+2.96 全 interest 策略贡献≈0","retro":"fin_cashflow 钦定 kind 集口径"}')
ON CONFLICT (kind,entity_ref) DO NOTHING;
