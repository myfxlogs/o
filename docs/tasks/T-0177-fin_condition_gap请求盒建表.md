# T-0177 fin_condition_gap 请求盒建表——D-0014 机制承载落地

## 态：可领

| 立项 | `[角色:决策]` @ 2026-10-27（债务盘点 A 级——D-0014 钉了机制、表从未迁；
业主接口面=唯一钦定交互径，现走会话） |
|---|---|

## 病灶

总纲「业主接口收窄=条件请求响应」的机制在 D-0014 钉为钦定位，但承载表
`fin_condition_gap` **从未建**——凭据缺/权限缺/条件缺时各臂只能打日志
（`[cmc] 凭据键缺——采集臂让位（业主条件请求件——对冲腿显读件缺）`
这类行就是孤儿请求），业主响应面无归口，系统也无法追踪「缺什么缺多久」。

## 施工面

1. 迁移件 `fin_0072`（0071 已被 T-0173 host_metrics 占）（fin 域——裁决/请托属钱径面非行情）：
   `fin_condition_gap(id bigserial PK, kind text NOT NULL CHECK
   (kind IN ('capital','credential','permission','constraint','holding',
   'executor')), venue text, surface text, payload jsonb NOT NULL
   DEFAULT '{}', state text NOT NULL DEFAULT 'pending' CHECK (state IN
   ('pending','fulfilled','rejected','dismissed')), requested_at
   timestamptz NOT NULL DEFAULT now(), resolved_at timestamptz,
   resolve_note text NOT NULL DEFAULT '', UNIQUE(venue,surface,kind)
   WHERE state='pending' 局部唯一)`——同面同类未决不重复开
2. 平台件 `internal/platform/condreq`（或归口既有窄包）：`Request(
   pool,kind,venue,surface,payload)` upsert 语义（pending 撞=刷新
   requested_at+payload 非重行）+`Resolve` 径
3. 接两处现存断点：`cmcArm` 凭据缺让位（trx.go）改发 `credential` 行；
   `fin_venue_untradeable`/venue 凭据检查同族位（票面允许最小两接点，
   全面接流后续票）
4. 读面：o-core ops face 已有多表读口径——挂只读查询面（前端面随 T-0173
   席次，不在本票）

## 验收锚

- 迁移过七断言闸；`fin_data_retention` 照账入册（不清账清道夫拒动）
- 桩锚：缺凭据态起 cmc 臂→`fin_condition_gap` 出 `credential/cmc` 行；
  补凭据后行 `fulfilled`（人工 UPDATE 口径可，自动关闭后续票）
- `check-env-reads`/`check_rejects_keys` 绿
