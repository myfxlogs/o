-- fin_0071 fin_condition_gap 请求盒（T-0177——D-0014 业主接口机制承载：
-- 凭据缺/权限缺/条件缺时各臂写请求行非孤儿日志；UNIQUE(venue,surface,kind)
-- WHERE pending=同面同类未决不重复开）。
CREATE TABLE IF NOT EXISTS fin_condition_gap (
    id           bigserial PRIMARY KEY,
    kind         text NOT NULL CHECK (kind IN ('capital','credential','permission','constraint','holding','executor')),
    venue        text,
    surface      text,
    payload      jsonb NOT NULL DEFAULT '{}',
    state        text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','fulfilled','rejected','dismissed')),
    requested_at timestamptz NOT NULL DEFAULT now(),
    resolved_at  timestamptz,
    resolve_note text NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS fin_condition_gap_pending_uq
    ON fin_condition_gap(venue,surface,kind) WHERE state='pending';
-- T-0153 保留账入册（清道夫拒动未入账表——permanent 裁决/请托域）。
INSERT INTO fin_data_retention(table_name,class,note)
VALUES('fin_condition_gap','permanent','请求盒裁决/请托面——永久保留')
ON CONFLICT (table_name) DO UPDATE SET class='permanent';
