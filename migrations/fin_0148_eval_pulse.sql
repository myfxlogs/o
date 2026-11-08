-- 0148 T-0485 eval 活动脉冲桥——per-template 新鲜度派生表（T-0479 裁量径②）。
-- 病灶：SG fin_opp_eval 镜像谓词=仅被 fin_proposal 引用行（D-0030 §三承重
-- 墙，PG-md ~93upd/s 体量闸不可全量镜像）→rejected-only 族 SG 侧结构性恒零
-- →族沉默探针+eval_zero 量表日频误报（六案同签名）；病灶=观测域错位非评估
-- 链 bug。本表=md 正本 fin_opp_eval per-template 聚合派生行（max(ts)/
-- count(24h)），projector serve 拍尾+活性对账顺带写 SG——量级=模板数
-- （~10 行），镜像体量可忽略。fin 链独表（md 侧无写者/读者不落正表——
-- 断言⑥ 桥表同形硬清单不在列）；domain.tsv 以 bridge 入册（md 域数据过桥
-- 语义面），消费面（族沉默探针/eval_zero 量表/活水段 eval 灯）活性判据
-- 改读 last_eval_ts 新鲜度——fin_opp_eval 镜像行数不再作 eval 活性判据。
CREATE TABLE IF NOT EXISTS md_eval_pulse (
    template     text        NOT NULL PRIMARY KEY,
    last_eval_ts timestamptz NOT NULL,
    eval_n_24h   integer     NOT NULL DEFAULT 0,
    refreshed_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE md_eval_pulse IS
  'eval 活性脉冲表（T-0485）——md fin_opp_eval per-template 聚合派生（projector serve 写）；脉冲行新鲜=族评估活，陈旧=真死——镜像谓词域外的活性判据面';
COMMENT ON COLUMN md_eval_pulse.last_eval_ts IS 'md 正本该 template max(ts)——活性判据唯一新鲜度源';
COMMENT ON COLUMN md_eval_pulse.eval_n_24h IS 'md 正本近 24h 行数（量表产出计数面——刷新间 staleness 以 last_eval_ts 为准）';
COMMENT ON COLUMN md_eval_pulse.refreshed_at IS 'projector 末次聚合写戳——脉冲面自身活性观测';

-- GRANT 幂等（fin_0099/0100 同款：角色缺席跳——测试实例零角色不炸；
-- projector_rw=写方生产在案，漏授=每拍 permission denied 实证坑）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'projector_rw') THEN
        EXECUTE 'GRANT SELECT,INSERT,UPDATE,DELETE ON md_eval_pulse TO projector_rw';
    END IF;
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'o_fin_ro') THEN
        EXECUTE 'GRANT SELECT ON md_eval_pulse TO o_fin_ro';
    END IF;
END $$;
