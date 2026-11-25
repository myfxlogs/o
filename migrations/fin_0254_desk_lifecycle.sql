-- 0254 票桌生命周期+积面终末消费者登记（工单O-39400——「只进不出」
-- 票桌病灶根修三件）：
-- ① fin_surface_lifecycle——「积累终态」面登记册：凡新建会积累行且有
--   终态语义的表/册必须登记 consumer_ref（谁消费它）+terminal_path
--   （怎么到终态）；check_all 增闸 check_surface_lifecycle.sh——新增
--   CREATE TABLE 无同迁移登记=闸红。初谱=存量五面人工册定注册物
--   （spec 钦定判据初谱）。
-- ② dm.hunger.desk_decay_days=14——票桌陈腐衰减阀窗（deadline 后 N 天
--   零活动→desk_stale 升件一次；hunger.ProbeDeskLifecycle fail-closed
--   键缺不探，落键防哑针）。
-- ③ ops_esc_valence——premise_dead/desk_stale 两新类惧值。
-- ④ 业主令#37640 封轴令物理补落——#38928 记「recommended→rejected
--   物理落地」但 fin_venue_candidate 实读全 recommended 零 rejected
--   （账实不符）；令原文=7户recommended翻rejected+hyperliquid免KYC
--   保留。此 UPDATE=令的物理执行补件（幂等 WHERE state 守卫——已翻
--   不重复动；反悔径=手工翻回 candidate 重测，令#37640 在案）。

CREATE TABLE IF NOT EXISTS fin_surface_lifecycle (
    surface       text PRIMARY KEY,
    accumulates   boolean NOT NULL,
    consumer_ref  text NOT NULL,
    terminal_path text NOT NULL,
    declared_at   timestamptz NOT NULL DEFAULT now()
);

INSERT INTO fin_surface_lifecycle(surface,accumulates,consumer_ref,terminal_path) VALUES
  ('fin_opp_template_roster', true, 'breedstall', 'hunger.ProbeBreedStall→升件立票'),
  ('fin_econ_lesson',        true, 'lessongate', 'lifecycle.declsion→admitProposals deadSigSuppress'),
  ('ops_ticket_desk',        true, 'CLOSESET',   'land_queue.sh 落成自动销+hunger.ProbeDeskLifecycle'),
  ('fin_venue_bill',         true, 'ingester',   'ledger/funding ingest_quote+tools/billprobe'),
  ('fin_cashflow',           true, 'recon',      'ledger reconcile→ledger_diff 升件')
ON CONFLICT (surface) DO NOTHING;

INSERT INTO app_config(key,value,updated_by) VALUES
  ('dm.hunger.desk_decay_days', '14', 'fin_0254')
ON CONFLICT (key) DO NOTHING;

INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('premise_dead', 60, '票桌前提死亡——venue判死直销/策略退役升件交裁（工单O-39400）'),
  ('desk_stale',    55, '票桌陈腐——deadline+N零活动单次升件（工单O-39400）')
ON CONFLICT (esc_key) DO UPDATE SET weight=EXCLUDED.weight, note=EXCLUDED.note;

UPDATE fin_venue_candidate SET state='rejected'
WHERE state='recommended'
  AND venue IN ('binance-usdm','bitget','bybit','dydx-v4','gate','kucoin-futures','mexc');

-- ⑤ sem_gate 放行面补宽（fin_0237 同型坑第二处实证）：落地件×处置
-- 矩阵行 note_like 原收窄「工单#% 销——落成自动销%」——desklife 探针
-- 销行（交付实据销/premise-dead）同属机制销行会被 BEFORE INSERT
-- trigger RAISE 拒写。放宽到「工单#% 销——%」=销——前缀本身就是
-- act_parse 的 close 词位（销[——：:，,+$]）——闸面与 act 语义对齐，
-- 落地件其余处置面（挂起/说明类）仍不放行，收窄性不变。
UPDATE ops_event_sem_matrix SET note_like='工单#% 销——%'
WHERE event='处置' AND seat='落地件';
