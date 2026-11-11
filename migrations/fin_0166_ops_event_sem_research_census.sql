-- 0166 研究席源普查落账——ops_event_sem_matrix 补「处置×研究席」放行组合。
-- 背景（升件#10368 实证）：research-intake-spec 钦定源普查登记账写径=
-- ops_event seat='研究席' event='处置' note='源普查: <源>→<过/毙>'
-- ——intake_dry 探针（cmd/core/supply_esc.go）即以 seat='研究席'
-- note LIKE '源普查%' 为活证口径;矩阵缺本对=spec 写径被闸拒,
-- 登记账物理不可落（2026-11-11 唤知会话手跑实证 RAISE——闸做对了:
-- 新 seat 未经裁处入册本就该拒）。业主令④放行进水口自主发现环+
-- 令⑦方向族入筛——登记账=覆盖率账本体。
-- 本迁只补放行一行（note_like 收窄到 源普查:% 前缀——研究席其余
-- 处置面不放行）;卡写面走 hypo 窄门不涉 ops_event。
-- 伴生补链：处置 事件经本对入闸集——存量放行对（处置席/复审席）系
-- 非迁径入册（fin_0154 基集不含处置行）,迁外差集使新 schema 与
-- 生产异形:测试库仅载本对→存量处置写全拒（check_all cmd/core
-- 12 测实证）。同迁补齐生产 处置 全集=闸语义新旧 schema 同形。
INSERT INTO ops_event_sem_matrix(event,seat,note_like,basis) VALUES
('处置','研究席','源普查:%','研究进水口源普查登记账（docs/drafts/research-intake-spec.md 钦定+supply_esc intake_dry 探针读面;每源一行覆盖率账）'),
('处置','处置席','','纪元3唤知会话处置写径（存量对迁外入册——本迁补链入 fin 链,new schema 与生产同形）'),
('处置','复审席','','纪元3独立复审会话处置写径（同上补链）')
ON CONFLICT (event,seat) DO NOTHING;
