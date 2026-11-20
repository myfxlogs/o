-- 1099 MT5 影子评估面（工单#25563 异因子数据面配套）：
-- fin_opp_template 登记 mt5_dir_mom——venue='mt5' 全目录（369 符 EXR 目录
-- 在流：美股CFD/指数/奇异FX/加密CFD）→md_klines venue='mt5' D1 棒喂入后
-- dirmom 泛 venue 臂（NewRunnerVenue）同判据影域评分。
-- 影子语义：execfam 不册 mt5_dir_mom——writeEval T-0330 同闸全落
-- rejected/exec_family_missing,eval 卡（z/sigma/net_apr）全可见、
-- fin_proposal 零到达=评分先行执行面零开通（T-0408 影子评估层先例）。
-- direction 词表复用 long_perp/short_perp（合法值零迁移——template_ref
-- 携 venue 语义；cfd 语义归属执行面工单另议，非本件域）。

INSERT INTO fin_opp_template(template,venue,surface,params) VALUES
  ('mt5_dir_mom','mt5','cfd_directional',
   '{"legs":["cfd_long","cfd_short"],"signal":"mom_z","settle":"mtm","shadow":true}')
ON CONFLICT DO NOTHING;
