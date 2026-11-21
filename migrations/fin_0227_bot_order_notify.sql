-- fin_0227_bot_order_notify.sql —— 经营推送面补盲（业主亲证劈差：手动
-- bot#1 开仓飞书零报——fin_bot_order 无 ev_* 触发器+EventPush 只扫
-- fin_position，bot 域开/停全静默）。INSERT=拉起帧，UPDATE OF state=
-- 停/killed 翻帧（快照同步只写 last_details 不触发，零噪声）。
-- 通道复用 ev_position（EventPush 已在听——不造新 NOTIFY 范式）。
CREATE OR REPLACE TRIGGER ev_bot_order_trg AFTER INSERT OR UPDATE OF state ON fin_bot_order
  FOR EACH ROW EXECUTE FUNCTION ev_notify('ev_position');
