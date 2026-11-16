-- 1097 eval-belief-insertonly fin 链（工单#19778 唤知裁定：版本化
-- insert-only——fin_opp_eval 自然键 UNIQUE 撤，写径改纯 INSERT 版本史，
-- 每评估一版 id 递增=入场信念成史可查；读面取最新=id DESC；
-- 桥表同形=md_1097 同文镜像（DDL 归一化断言钉——两链语句全等）。
-- upsert 灭史病灶实案：-14.4% 预测实为入场后重估 -34%（幻影——旧行被覆写
-- 拿不到入场信念）。消费面 latest 语义：md 侧读径逐点核（volsell/basis
-- eval 引用取 id DESC；eval 脉冲 count 改 DISTINCT 自然键）；SG 镜像侧
-- 保单行/键（uEval 手工 upsert——最新 ts 赢、id 稳），消费面零改动。
ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_key;
CREATE INDEX IF NOT EXISTS fin_opp_eval_key_latest
    ON fin_opp_eval(inst_id, template, direction, id DESC);
