-- 1068 app_config 第九桥 PG-md 只读副本（T-0169 承接段裁：投影第九桥——
-- 分域后 collector/research/projector config.Load 读本池，正本仍 SG 写
-- （app_config 归 fin 不变），projector 听 SG config_changed 全量 diff 副本
-- +回灌 md config_changed（热启链路保全）。app_secrets 不副本（密文域不出 SG）。
-- DDL 与 fin 链 fin_0001 正表归一化同形（桥表断言钉）；触发器/代理码不在副本
-- （写方唯一=projector，generation/updated_at 随行verbatim搬运）。
CREATE SEQUENCE IF NOT EXISTS config_generation_seq;

CREATE TABLE IF NOT EXISTS app_config (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL,
    generation bigint NOT NULL DEFAULT nextval('config_generation_seq'),
    updated_at timestamptz NOT NULL DEFAULT now()
);
