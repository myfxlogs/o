-- 1080 T-0298 basis 两模板台账归位（D-0043/CTO 会签件①重定性：enabled 非功能
-- 闸——paperTemplates 白名单早已在册，翻 true=名册真化非解封；ops_event 留痕
-- 在 fin_0097）。quarter_basis_short 保禁不动（spot_short 腿缺 margin 面）。
UPDATE fin_opp_template SET enabled=true, state='enabled'
  WHERE template IN ('quarter_basis_long','perp_quarter_spread');
