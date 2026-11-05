-- 0112 T-0353 件①中间档产线阈两级化（业主裁 B——打包件
-- docs/cos/packs/2026-11-05-middle-band-production-line.md 附钉①）：
--   近确定 carry 族（quarter_basis_long/perp_quarter_spread/dcd_call/dcd_put）
--   降档 0.04（4.84% 期差基差 net_apr 被纯阈错杀根修）；funding-hedge 及
--   未列族保 0.06 纯阈档不变。阈存址=既有 per-family config 键（评估器
--   原读径零改——族级映射显式枚举=internal/research/band 注册表+boot 审计）。
--   ②vol-sell 两族与 dcd 阈键解耦：原 T-0215 复用 research.dcd.min_apr
--   （0.05）——dcd 族降 0.04 会连带卖权族越白名单（directional 不进中间档
--   红线），自持键归位 Default 档。③research.dcd.fail_streak_n=复役条件书
--   机器判据（连败 N 拍自停+升件——件③）。④fin_condition_gap INSERT 补授
--   research_rw（T-0177 设计本义「各臂写请求行」——research 节点分迁后
--   授权漏项，件④业主资产域分流写径前置件；限列授予请求刷新和复役状态读取，
--   不授 state 更新权，业主处置仍走既有窄写面）。
UPDATE app_config SET value='0.04',updated_at=now(),updated_by='fin_0112'
  WHERE key='research.basis.min_net_apr';
UPDATE app_config SET value='0.04',updated_at=now(),updated_by='fin_0112'
  WHERE key='research.dcd.min_apr';

INSERT INTO app_config(key,value) VALUES
  ('research.volsell.min_apr', '0.06'),
  ('research.dcd.fail_streak_n', '3')
ON CONFLICT (key) DO NOTHING;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
    GRANT INSERT ON fin_condition_gap TO research_rw;
    GRANT SELECT (id,kind,venue,surface,payload,state,requested_at) ON fin_condition_gap TO research_rw;
    GRANT UPDATE (requested_at,payload) ON fin_condition_gap TO research_rw;
    GRANT USAGE ON SEQUENCE fin_condition_gap_id_seq TO research_rw;
  END IF;
END $$;

INSERT INTO ops_event(seat,event,ticket,note) VALUES
  ('施工席','启用','T-0353',
	'阈两级化落键：basis/dcd 族 0.04（近确定 carry 档）、volsell 解耦自持 0.06、funding 保 0.06；fail_streak_n=3（严格 >N）复役条件书判据；fin_condition_gap 请求刷新+复役状态查询列授 research_rw——不授 state 写权；业主裁B附钉①②③随键生效');
