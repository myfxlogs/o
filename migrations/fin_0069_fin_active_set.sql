-- 0069 fin_active_set 活跃集发布表（T-0167/D-0030 §四：活跃集谓词由 SG 发布——
-- fin_position 未平仓 ∪ fin_strategy 非 retired trigger 维护 inst_id 集，
-- 行变 NOTIFY ev_active_set；projector（T-0168）订阅其帧刷投影谓词——
-- D-0008 零轮询律，弃「读一次/分钟」。名 md_active_set 实 fin——建表即 fin_ 前缀
-- （D-0030 §五残名清口径，列残名不再新增）。
CREATE TABLE IF NOT EXISTS fin_active_set (
    inst_id    text PRIMARY KEY,
    updated_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_active_set IS
  '活跃集发布表（D-0030 §四）——在仓 fin_position ∪ 非 retired fin_strategy 的 inst_id 集，projector 投影谓词源';

-- inst 级重算维护：任一证据成立=在集，否则出集；行变即 NOTIFY（纯唤醒，
-- payload 空=判定数据现读惯例同 0028 事件脊）。
CREATE OR REPLACE FUNCTION fin_active_set_maintain() RETURNS trigger AS $$
DECLARE
    inst text;
BEGIN
    IF TG_OP = 'DELETE' THEN
        inst := OLD.inst_id;
    ELSE
        inst := NEW.inst_id;
    END IF;
    IF EXISTS (SELECT 1 FROM fin_position p
                 WHERE p.inst_id = inst AND p.closed_at IS NULL)
       OR EXISTS (SELECT 1 FROM fin_strategy s
                    WHERE s.inst_id = inst AND s.state <> 'retired') THEN
        INSERT INTO fin_active_set(inst_id) VALUES (inst)
            ON CONFLICT (inst_id) DO UPDATE SET updated_at = now();
    ELSE
        DELETE FROM fin_active_set WHERE inst_id = inst;
    END IF;
    PERFORM pg_notify('ev_active_set', '');
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER fin_active_set_position_trg
  AFTER INSERT OR UPDATE OR DELETE ON fin_position
  FOR EACH ROW EXECUTE FUNCTION fin_active_set_maintain();
CREATE OR REPLACE TRIGGER fin_active_set_strategy_trg
  AFTER INSERT OR UPDATE OR DELETE ON fin_strategy
  FOR EACH ROW EXECUTE FUNCTION fin_active_set_maintain();

-- 保留账入册（T-0153 清道夫 fail-closed）：inst_id PK 触发器维护集=state 不删行
-- （键域封顶=活跃 inst 数，出集即 DELETE）。
INSERT INTO fin_data_retention(table_name,class,note)
VALUES ('fin_active_set','state','状态表不删行——活跃集发布面（projector 谓词源）')
ON CONFLICT (table_name) DO NOTHING;
