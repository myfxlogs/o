-- 0145 T-0458 飞书席位活卡——ops_seat_card 卡册（seat PK→message_id 持久）。
--   一席一卡（业主裁：按席开不按票开——看的=席位活性非票级流水）：
--   seat→message_id+chat_id 落 PG，重启原地刷同卡（meta feishu_card 同构
--   先例——card.go 三件套模式级复用：内容哈希闸 PATCH/懒建/启动对账）；
--   last_hash=卡内容 sha256 hex（哈希闸重启续命——重启不重发无变卡）。
--   卡=展示面非真源（真源=ops_event append-only 链账）；卡败落 ops_event
--   seatcard_err 行——非 eventbus dm 帧白名单事件，不回灌自触发环（meta
--   RecordErr 同判例）。
--   写径=FIN_PG_DSN（roster_rw——ops_event 同族写册角色）。
CREATE TABLE IF NOT EXISTS ops_seat_card (
	seat       text PRIMARY KEY,
	message_id text NOT NULL DEFAULT '',
	chat_id    text NOT NULL DEFAULT '',
	last_hash  text NOT NULL DEFAULT '',
	updated_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE ops_seat_card IS '飞书席位活卡卡册（T-0458——seat→message_id 持久映射按席一卡；展示面非真源，真源=ops_event）';
COMMENT ON COLUMN ops_seat_card.last_hash IS '上次成功落卡内容 sha256 hex（哈希闸重启续命——同内容零 PATCH）';

-- 写径授权（fin_0121 同族规——roster_rw/research_rw=ops_event 写册角色，
-- 角色缺席环境=零失败；卡册写径=seatcard 服务 FIN_PG_DSN 同源）。
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
    GRANT SELECT,INSERT,UPDATE ON ops_seat_card TO roster_rw;
  END IF;
  IF EXISTS (SELECT FROM pg_roles WHERE rolname='research_rw') THEN
    GRANT SELECT,INSERT,UPDATE ON ops_seat_card TO research_rw;
  END IF;
END $$;
