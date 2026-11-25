-- fin_0259_ledger_rebase.sql —— O-33206 业主令账本重锚武装键：
--
-- 业主令 2026-11-25「以前的旧账都不要了,以昨天的账为基点」——
-- internal/ledger/rebase.go rebaseDecree：armed 在场→scan 拍先于
-- reconcile 跑，对拍行集（reconRows 同源）逐键 delta=venue实读−
-- 账本净 落 recon_adjust(account_id 直发,source_table=
-- 'rebase_owner_decree'+source_ref=令簿记 ops_event 行)，幂等=
-- 桶级已有重锚行跳+跑毕置 'done' 单次性；静态 USD 残差退役
-- （resolver 跟踪型保留——构性漂移差会再生）。
-- 本迁移只落武装键——重锚行本体=运行时 venue 实读锚（禁猜，迁移
-- 不写死金额）；读败/在途窗=armed 保留下拍重试，盲锚禁。

INSERT INTO app_config(key,value,updated_at,updated_by)
VALUES('ledger.rebase_armed','"armed"',now(),'migration/fin_0259')
ON CONFLICT(key) DO NOTHING;
