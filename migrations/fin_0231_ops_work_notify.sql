-- fin_0231 工作面事件脊 ev_work（工单O-32300——拍边界等待病灶根修：
-- ops_event 关键行落库即 NOTIFY 唤醒对应 dispatch，15min timer 降级为
-- 漏唤看门狗双保险；判定数据一律现读，payload 恒空=纯唤醒语义）。
-- 行级 WHEN 收窄防 notify 洪泛（fin_0185/fin_0028 同病灶律），覆盖：
--   工单      ——票落板即路由（求知→research/普通→worker/管内→cognition）；
--   升件      ——非 crit 类直唤唤知（crit 已有 mechwake dm 径，本件对齐扩面，
--                判据阈值归 dispatch.sh 内部不在脊层）；
--   完工回执/候审回执/复审过 ——候审链两跳（回执→唤知复审验收；
--                复审过→land_queue 机械合），消「滞留警报→等拍」一跳；
--   note 词面兜底 ——记事/处置等异 event 行携回执词面同收（双写一致件：
--                谓词与 scripts/cognition/wakebus.sh 读行谓词同文，改须同改）。
CREATE OR REPLACE TRIGGER ev_work_trg
  AFTER INSERT ON ops_event
  FOR EACH ROW
  WHEN (NEW.event IN ('工单','升件','完工回执','候审回执','复审过')
        OR NEW.note ~ '完工回执|完工推支候审|候审回执|复审过[[:space:]]+sha=')
  EXECUTE FUNCTION ev_notify('ev_work');

COMMENT ON TRIGGER ev_work_trg ON ops_event IS
  '工作面唤醒脊→ev_work一跳(工单/升件/候审链关键行——行级WHEN收窄防洪泛)';
