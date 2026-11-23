-- md_1106 OSS 三族骨架模板行（工单O-35489——假想卡#578380-82 parked
-- 语义正本逐族五件落位②）：fin_opp_template 登记三族 candidate 态
-- （未收养非钦定禁用——grammar upsert 同形；族 Existing=true 文法不重生
-- 行/假想卡/评估，假想卡 adopt=席权 decide 窄门另件本迁移不写）。
-- params=腿原语（卡 leg_primitives 载）+信号/结算快照+判据键前缀指引
-- （判据真值=fin 链 fin_0241 app_config research.<fam>.*——卡
-- falsify/ev_formula 可算口径机读，禁虚构常数）。
-- venue='okx' 全族——数据面 md_klines/md_funding/md_ticks/md_mark_index
-- 在册（spec 钦定已验齐）。

INSERT INTO fin_opp_template(template,venue,surface,params,enabled,state,
  disabled_reason,ban_premise_kind) VALUES
  ('xsec-perp-mom-basket','okx','xsec-perp-mom-basket',
   '{"legs":["perp_long_basket","perp_short_basket"],"signal":"xsec_mom_basket","settle":"mtm","rebal":"monthly","kind":"xsec_mom","hypothesis":578380,"criterion":"research.xsec_mom.*"}',
   false,'candidate','未收养候选——非钦定禁用','none'),
  ('funding-extreme-fade','okx','funding-extreme-fade',
   '{"legs":["perp_fade_dyn"],"signal":"funding_p95_fade","settle":"mtm","kind":"mean_reversion","hypothesis":578381,"criterion":"research.funding_fade.*"}',
   false,'candidate','未收养候选——非钦定禁用','none'),
  ('mark-last-disloc-fade','okx','mark-last-disloc-fade',
   '{"legs":["perp_fade_dyn"],"signal":"mark_last_disloc_fade","settle":"mtm","kind":"mean_reversion","latency":"budget_gate","hypothesis":578382,"criterion":"research.disloc_fade.*"}',
   false,'candidate','未收养候选——非钦定禁用','none')
ON CONFLICT (template,venue,COALESCE(surface,'')) DO NOTHING;
