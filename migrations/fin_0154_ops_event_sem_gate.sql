-- 0154 T-0506 ops_event 写面语义闸——(seat,event) 合法性矩阵落库执行。
-- 病灶（2026-11-09 实证）：相位投影 seat 宽（ops_ticket_phase 只看 event
-- 词）× 消费面 seat 窄（dm_queue dispatched 面滤 latest_seat∈{决策席,CTO}）
-- 断层——COS 清账批 seat='COS' 写派单 5 票→票呈 dispatched 而补拉窗全盲
-- 4h 零拉起（条目 dispatch-seat-whitelist-phase-lie 在案）。键形闸
-- （T-0499）只管 ticket 形不管组合合法性；裸 psql INSERT 一切写径闸全绕
-- ——闸必须落库执行层（BEFORE INSERT trigger）不靠写径自觉（fin_0121
-- hash 链同判据：写面异构多径无单一 app 层收口点，库内闸才全径必经）。
--
-- 结构两件：
--   ① ops_event_sem_matrix 白名单表——(event,seat,note_like,basis)；
--     note_like=''=任意 note，非空=NEW.note LIKE 判定（cos-relay 例外径）。
--     表内只载「相位驱动词×seat」子集（先钉子集全量收紧留观察窗——
--     票面判据②）；矩阵外事件=不设闸宽容放行。
--   ② ops_event_sem_gate() BEFORE INSERT trigger——矩阵内 event 且
--     (seat,note) 无命中行→RAISE EXCEPTION loud 拒（ERROR 明文
--     seat/event/ticket 三元——票面判据②）；非矩阵 event 直通。
--
-- 矩阵内容判据（禁线：消费面判据+存量组合为准绳，禁凭直觉添删 seat）：
--   派单  = dm_queue.sh dispatched 面钦定集 {决策席,CTO}（消费面 SQL
--           逐字）+ COS 席族×note~cos-relay 例外（T-0329 钦定转办标——
--           COS 职卡写面白名单「派单（cos-relay 标）」；COS/COS上行/
--           席O-COS 三变体同豁免）；其余 seat 写派单=相位说谎病灶本体
--           一律拒（清账批/内联径误用/测试行/死席别名全拒）。
--   裁决/驳回 = 票面例 {决策席,CTO,裁定席,CEO,外审席} + 存量在产变体
--           {外审(外审席别名在产 2026-11-09),COS(collab 窄口豁免兜底裁
--           +存量),席O-COS(COS族变体业主授权直裁提案件),审计席(存量)}；
--           verdict_deliver 消费面 {决策席,CTO} 只管需业主动作判词送达
--           不构成写面全集；ops_decision_mirror 全 seat 镜像不受本闸影响
--           （镜像 AFTER INSERT 只见落库行——拒写=镜像同拒语义自洽）。
--           施工席族/死席别名（Dev/DM/席O-DM/席MANAGE 等）写裁决驳回=
--           越权拒——无头施工席误标裁决行（11-08 批拍实证）自此 loud 拒。
--   领单  = 票面例「施工席族」——施工席+无头施工席（report.sh 产名双
--           轨）+存量在产交互/会话变体 {O-Dev交互席,席O-Dev,席O-DM,
--           席Claude,席Devin,席claude,claude-kimi,席MANAGE}；决策席/
--           CTO/DM/Dev 写领单=票被非施工席冒领拒（存量 5 组合在拒列）。
--   呈批  = ceo_queue.sh 消费面席集逐字（席集随消费面漂移——T-0507
--           MECH 层扫席入册后 14 席；呈批入 CEO 队列的唯一通道——
--           集外 seat 写呈批=队列盲区说谎，席O-COS 存量 5 行即此型
--           拒写倒逼改署 COS上行/COS）。
--   已批  = 批准权写径——存量在产 {决策席,CTO,CEO(dm_autoapprove 机批
--           钉死 seat='CEO'),COS(职卡例行代理批注钦定),O-Dev交互席}；
--           施工席族写已批=自批自派拒（approved 面零 seat 滤→错批=错派
--           真病灶）。
--   起草  = 存量全集 {决策席,CTO}——drafterSeat 钉死决策席；起草行=
--           dm_autoapprove 候选注入键，错席起草=误注入机批流拒。
--   拉起  = 存量全集 {决策席,CTO,META-DM}——拉起=票离 dispatched 队
--           假拉起=静默尸（dm_handoff 接行对 seat 无感），错席拒。
--
-- 不设闸相位词（宽容放行——消费面零 seat 判据，相位真值与 seat 无关，
-- 存量组合全活跃）：开工/外派/收口/移交/收口待审/审计中/完成/挂起/
-- 作废/异常。说明类事件（升件/处置/候水/送达/退班/领号/归流/部署/
-- 诊断草案/议题/通过/施工中/COS上行/业主令/业主批/管理令/评审拍/转办/
-- 等锁/先例适用/消/新品/达阈/启用/领取/返修/已立/并票/异常词/开仓挂起/
-- 开仓解冻/裸袋/杂币敞口/价格提醒/宏观提醒/feed断更/心跳/教训矿工/
-- 族沉默/覆盖失明/依赖对清单/依赖对失明/依赖对豁免/饥饿/饥饿响应/
-- 面清单新面/漂移/漂移消/账链断/GC/seatcard_err/earn_auto_purchase/
-- price_alert_build/price_alert_state/quote_watchlist_set/优先令/草案/
-- 闸值钦定/冲销迹/勘正/待部署/部署中/已并码 等一切非相位词）天然放行。
--
-- 存量组合全覆盖对拍（判据④）：矩阵对在产 DISTINCT(seat,event) 全谱
-- 分类完备——闸内 8 词×在产组合逐一有判（放行/条件放行/拒写三态），
-- 拒列组合=死席别名+病灶写+越权写（scripts/meta/tests/
-- ops_event_sem_gate_test.sh B 段枚举对拍钉死，新增存量组合入闸=测试
-- 红=强制回本矩阵对账——漏枚举自残由此闸住）。
CREATE TABLE IF NOT EXISTS ops_event_sem_matrix (
	event     text NOT NULL,
	seat      text NOT NULL,
	note_like text NOT NULL DEFAULT '', -- ''=任意 note；非空=NEW.note LIKE 判定
	basis     text NOT NULL,
	PRIMARY KEY (event, seat)
);
COMMENT ON TABLE ops_event_sem_matrix IS 'ops_event (seat,event) 合法性矩阵（T-0506——相位驱动词×seat 白名单；表外事件不设闸；note_like 空=任意 note 非空=LIKE 条件放行）';

INSERT INTO ops_event_sem_matrix(event,seat,note_like,basis) VALUES
-- 派单：消费面钦定集+COS席族cos-relay例外（T-0329）
('派单','决策席','', 'dm_queue dispatched面 seat∈{决策席,CTO} 消费面SQL逐字'),
('派单','CTO','','dm_queue dispatched面+dm_depgate/mech_tick 壳轨写径'),
('派单','COS','%cos-relay%','T-0329钦定转办标——COS职卡写面白名单'),
('派单','COS上行','%cos-relay%','同上 COS席位变体'),
('派单','席O-COS','%cos-relay%','同上 O-COS tmux会话变体（T-0474入册）'),
-- 裁决/驳回：票面例集+存量在产变体（verdict_deliver消费面不构成写面全集）
('裁决','决策席','','票面例+collab写面判据裁决行属决策席+存量在产'),
('裁决','CTO','','席名归一别名同上'),
('裁决','裁定席','','票面例钦定——案裁 verbatim'),
('裁决','CEO','','票面例+ceo_tick升级裁存量在产'),
('裁决','外审席','','票面例+外审席职卡外派归宗裁存量在产'),
('裁决','外审','','外审席别名变体存量在产（2026-11-09仍写）'),
('裁决','COS','','collab窄口豁免兜底裁+存量在产'),
('裁决','席O-COS','','COS族变体存量在产（业主授权直裁提案件）'),
('裁决','审计席','','存量组合在案'),
('驳回','决策席','','裁决同消费面族（verdict_deliver/镜像/队列终态）'),
('驳回','CTO','','同上'),
('驳回','裁定席','','同上'),
('驳回','CEO','','同上'),
('驳回','外审席','','同上'),
('驳回','外审','','同上'),
('驳回','COS','','同上'),
('驳回','席O-COS','','同上'),
('驳回','审计席','','同上'),
-- 领单：票面例施工席族+存量在产变体
('领单','施工席','','票面例施工席族+report.sh映射产名'),
('领单','无头施工席','','票面例施工席族+dispatch_worker缺省产名'),
('领单','O-Dev交互席','','存量组合在产——交互施工席变体'),
('领单','席O-Dev','','存量组合在产'),
('领单','席O-DM','','存量组合在产'),
('领单','席Claude','','存量组合在产——harness会话名'),
('领单','席Devin','','存量组合在产'),
('领单','席claude','','存量组合在产'),
('领单','claude-kimi','','存量组合在产'),
('领单','席MANAGE','','存量组合在产'),
-- 呈批：ceo_queue.sh 消费面席集逐字 13 席
('呈批','COS上行','','ceo_queue消费面席集逐字'),
('呈批','COS','','同上'),
('呈批','O-Dev交互席','','同上'),
('呈批','决策席','','同上'),
('呈批','META-DM','','同上'),
('呈批','CTO','','同上'),
('呈批','CEO','','同上'),
('呈批','裁定席','','同上'),
('呈批','施工席','','同上'),
('呈批','外审席','','同上'),
('呈批','CEO上行','','同上'),
('呈批','管理席','','同上'),
('呈批','研究员','','同上'),
('呈批','MECH','','同上——T-0507 层扫 digest 席入册 ceo_queue 席集'),
-- 起草/拉起：存量全集钉死（drafterSeat=决策席；拉起=离 dispatched 队键）
('起草','决策席','','存量全集——drafterSeat 钉死决策席；起草行=dm_autoapprove 候选注入键'),
('起草','CTO','','同上'),
('拉起','决策席','','存量全集——拉起=票离 dispatched 队，假拉起=静默尸'),
('拉起','CTO','','dm_depgate/mech_tick/dm_suspend 壳轨写径'),
('拉起','META-DM','','存量在产组合'),
-- 已批：批准权写径（错批=错派真病灶——approved面零seat滤）
('已批','决策席','','存量在产组合——决策裁断→已批'),
('已批','CTO','','同上'),
('已批','CEO','','dm_autoapprove机批钉死 seat=CEO'),
('已批','COS','','COS职卡例行代理批注钉定+存量在产'),
('已批','O-Dev交互席','','存量在产组合')
ON CONFLICT (event,seat) DO NOTHING;

-- 语义闸函数 BEFORE INSERT：矩阵内 event 且 (seat,note) 无命中→loud 拒。
-- 拒写=RAISE EXCEPTION（ERROR 明文 seat/event/ticket 三元——裸 psql/
-- report.sh/Go 服务族/壳脚本全径必经同拒）；非矩阵 event 直通零开销。
CREATE OR REPLACE FUNCTION ops_event_sem_gate() RETURNS trigger AS $$
BEGIN
	IF EXISTS (SELECT 1 FROM ops_event_sem_matrix m WHERE m.event = NEW.event) THEN
		IF NOT EXISTS (
			SELECT 1 FROM ops_event_sem_matrix m
			WHERE m.event = NEW.event AND m.seat = NEW.seat
			  AND (m.note_like = '' OR NEW.note LIKE m.note_like)
		) THEN
			RAISE EXCEPTION 'ops_event_sem_gate: 拒写非法 (seat,event)——seat=% event=% ticket=%（合法性矩阵 fin_0154；相位驱动词×seat 白名单落库执行，说明类/矩阵外事件不受闸）',
				NEW.seat, NEW.event, NEW.ticket;
		END IF;
	END IF;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
COMMENT ON FUNCTION ops_event_sem_gate() IS 'ops_event (seat,event) 语义闸（T-0506——矩阵内 event 无命中行 loud 拒；矩阵外事件直通）';

-- 触发器名序在 ops_event_chain_trg 后（字母序 c<s）——拒写行链戳白算
-- 无害（事务随 RAISE 中止，advisory 锁 xact 级随回滚放）。
DROP TRIGGER IF EXISTS ops_event_sem_gate_trg ON ops_event;
CREATE TRIGGER ops_event_sem_gate_trg BEFORE INSERT ON ops_event
	FOR EACH ROW EXECUTE FUNCTION ops_event_sem_gate();

-- 授权面：矩阵表=闸判据读面——写角色（research_rw/roster_rw/review_rw
-- 持 ops_event INSERT）须持 SELECT 否则触发器 INVOKER 语义插行炸
-- （fin_0121 链头读同律）；角色缺席环境零炸同族规。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='research_rw') THEN
		EXECUTE 'GRANT SELECT ON ops_event_sem_matrix TO research_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT ON ops_event_sem_matrix TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='review_rw') THEN
		EXECUTE 'GRANT SELECT ON ops_event_sem_matrix TO review_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON ops_event_sem_matrix TO o_fin_ro';
	END IF;
END $$;
