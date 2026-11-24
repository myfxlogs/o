package pgstore

// —— md 链（版本链内唯一且单调；md 链版本=历史号+1000 与 fin 键空间全局不相交——跨链并 schema 时 schema_migrations 零撞号，check_migrations 断言①⑤）——
var mdMigrationTable = []migEntry{
	{1003, "md_1003_market_facts.sql"},
	{1006, "md_1006_research.sql"},
	{1011, "md_1011_review.sql"},
	{1015, "md_1015_earn.sql"},
	{1016, "md_1016_earn_rate.sql"},
	{1020, "md_1020_macro_event.sql"},
	{1028, "md_1028_event_triggers.sql"},
	{1039, "md_1039_next_funding.sql"},
	{1044, "md_1044_capability_surface_dcd.sql"},
	{1045, "md_1045_hypothesis_chain.sql"},
	{1049, "md_1049_md_state_tables.sql"},
	{1051, "md_1051_capability_dense.sql"},
	{1052, "md_1052_grammar_candidates.sql"},
	{1053, "md_1053_eval_state_table.sql"},
	{1054, "md_1054_proposal_dedupe_gate.sql"},
	{1056, "md_1056_proposal_eval_id_idx.sql"},
	{1057, "md_1057_md_signal.sql"},
	{1060, "md_1060_xvenue_carry.sql"},
	{1061, "md_1061_margin_borrow_capability.sql"},
	{1063, "md_1063_data_retention.sql"},
	{1064, "md_1064_notify_flood_shrink.sql"},
	{1065, "md_1065_proposal_retire_trace.sql"},
	{1066, "md_1066_md_funding_uq.sql"},
	{1067, "md_1067_projector_frames.sql"},
	{1068, "md_1068_app_config_replica.sql"},
	{1069, "md_1069_data_retention_md.sql"},
	{1070, "md_1070_fin_strategy_replica.sql"},
	{1071, "md_1071_snapshot_price_retire.sql"},
	{1072, "md_1072_app_config_audit.sql"},
	{1073, "md_1073_ev_ticks_trg_restore.sql"},
	{1074, "md_1074_basis_direction_defork.sql"},
	{1075, "md_1075_arch_eval.sql"},
	{1076, "md_1076_arch_signal.sql"},
	{1077, "md_1077_retire_reason_forced.sql"},
	{1078, "md_1078_eval_proposed_net_apr_gate.sql"},
	{1079, "md_1079_hypothesis_parked.sql"},
	{1080, "md_1080_basis_template_enable.sql"},
	{1081, "md_1081_venue_funding.sql"},
	{1082, "md_1082_venue_candidate.sql"},
	{1083, "md_1083_ctx_regime.sql"},
	{1084, "md_1084_ctx_dedupe.sql"},
	{1085, "md_1085_venue_measuring.sql"},
	{1086, "md_1086_stake_carry.sql"},
	{1087, "md_1087_stake_carry_fin_replica.sql"},
	{1088, "md_1088_venue_candidate_ibkr.sql"},
	{1089, "md_1089_volsell_capability_reenable.sql"},
	{1090, "md_1090_venue_onboard_states.sql"},
	{1091, "md_1091_two_tier_collect.sql"},
	{1092, "md_1092_scan_universe.sql"},
	{1093, "md_1093_repair_log.sql"},
	{1094, "md_1094_klines.sql"},
	{1095, "md_1095_dir_perp.sql"},
	{1096, "md_1096_perp_trend_daily.sql"},
	{1097, "md_1097_eval_insertonly.sql"},
	{1098, "md_1098_trend_template_park.sql"},
	{1099, "md_1099_mt5_dir_mom.sql"},
	{1100, "md_1100_gridfit.sql"},             // 工单#25737：网格族影子评估面（fin_0215 值 CHECK 放行+模板册行）
	{1101, "md_1101_opt_chain.sql"},           // 工单#32540：OKX 期权链采集面（instruments/summary/ticker/snapshot 四表——covered-call 数据前置）
	{1102, "md_1102_trend_mech_card.sql"},     // 工单O-32604：假想卡② trend_mech 落账（proposed——方向族死刑审查输入件）；md_1101 双占撞号改钉 1102
	{1103, "md_1103_tmpl_ban_fields.sql"},     // 工单O-33393：fin_opp_template 禁因三件套+名册视图（正本=PG-md——enabled=f 禁因机读化+state 矛盾态清零+candidate/禁用分态）
	{1104, "md_1104_dir_perp_trend.sql"},      // 工单O-35491②：direction 词表扩 perp_trend（fin_0240 对偶——两链 ALTER 归一化全等）
	{1105, "md_1105_md_oi_liq.sql"},           // 工单O-35587②/#35490：md_open_interest OI 状态表+md_liquidation 清算事件流水（超表90d+天然键去重——两表入 fin_data_retention 账）
	{1106, "md_1106_oss_fam_skeleton.sql"},    // 工单O-35489：OSS 三族骨架模板行（卡#578380-82——candidate 态未收养；md_1104/1105 在途占跳号钉 1106）
	{1107, "md_1107_mt5_klines_d1_purge.sql"}, // 工单O-38593：mt5 '1D' 周棒误标清污（timeFrame 分钟制纠偏——dow=0 谓词只削周签名行）
}
