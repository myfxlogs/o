-- fin_0148 —— GW-001 段D 诊断前段两件：
--   ① guard.postmortem.fast_close_repeat_n 热启键种子（段D1——fast_close
--      复发闸 N：7d 窗同 strategy_ref∨inst_id 含本仓共 ≥N 次∨settled<0
--      才入诊断闸，单次秒平不烧诊断额度。N=2=第二次复发即立案；
--      guard.* 族 DM 写域——dmconfig 调阈径照旧）。
--   ② fin_ticket_draft pending 草案防重唯一索引（段D4 幂等实证面——
--      归流执行双执行者并存期（新 verdict_reflow.sh × 在飞裁定席旧
--      SKILL 归流段）防双归流重复出 draft 的 DB 级硬闸：同
--      (signature,lesion_key) 至多一张 pending 草案。kind 收窄
--      rootcause_diag/trade_postmortem 两族（段D 触碰面）——esc 探针族
--      历史数据不入索引面不赌存量形态；已 materialized/dismissed 行
--      出索引（同族 lesion 复发再立新草案合法径不受阻）。
--      并存期语义：后到 INSERT 撞唯一=整语句败退，先到者赢——两执行者
--      各自的翻案 UPDATE 另有 status='drafted' CAS 兜底，先到翻案者赢，
--      后到者按翻案败退径走归流行补写，零重复零丢失）。

INSERT INTO app_config (key, value) VALUES
  ('guard.postmortem.fast_close_repeat_n', '2')
ON CONFLICT (key) DO NOTHING;

CREATE UNIQUE INDEX IF NOT EXISTS fin_ticket_draft_pending_dedup
  ON fin_ticket_draft (signature, lesion_key)
  WHERE status = 'pending'
    AND kind IN ('rootcause_diag', 'trade_postmortem');
