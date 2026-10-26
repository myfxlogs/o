-- 0002 花名册（称谓层 D-0003：台账实体=员工，fin_employee_registry=花名册）。
-- 运行时本体 phase-core=1 号员工：boot 自注册 Register + 周期 Beat。
CREATE TABLE IF NOT EXISTS fin_employee_registry (
    name          text PRIMARY KEY,
    dept          text NOT NULL,
    title         text NOT NULL DEFAULT '',
    status        text NOT NULL DEFAULT 'active',
    meta          jsonb NOT NULL DEFAULT '{}',
    registered_at timestamptz NOT NULL DEFAULT now(),
    last_beat     timestamptz NOT NULL DEFAULT now()
);
