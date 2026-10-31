# T-0281 SG fin_capability_surface okx/dcd 残留钉 f（T-0276 审计留案①）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-10-31（MANAGE-3414③ 裁决——审计注记②留案一「是否顺带钉」裁=钉，单列小票走流程） |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | 低（无消费者——潜伏口径歧离件，非功能性病灶） |
| 涉及面 | SG `o-postgres` `fin_capability_surface`（单行 UPDATE，无码改） |

## 背景

T-0276 dcd 退役三钉落了 PG-md 面（`fin_capability_surface` okx/dcd=f 在案），
但 SG `o-postgres` 上的 fin schema 独立副本残留 `enabled=t`（审计实查）。
消费面实况：枚举臂/grammar/gapscan 等读方全在 OCI 侧读 PG-md 正本=f——SG
副本无消费者，退役态语义不受影响（审计注记②原文）。

歧离本身是坑：SG=正本与 PG-md=副本对同一面报相反口径，任何未来 SG 侧读方
/全量重建/人肉对拍都会被 t 残留误导（「dcd 仍启用」假象）。钉 f 是一行
UPDATE 的对齐成本，消一类歧义。

## 目标

SG `o-postgres` `fin_capability_surface` `venue='okx' AND surface='dcd'` →
`enabled=false`，与 PG-md 退役态对齐；其余行零动。

## 验收标准

- [ ] SG `docker exec o-postgres psql -U o -d o -c "SELECT enabled FROM fin_capability_surface WHERE venue='okx' AND surface='dcd'"` 实查=f
- [ ] 过一个副本同步周期后复拍：SG 仍 f、PG-md 仍 f（无双侧回弹）
- [ ] `dcd-test` 行与其余 surface 行零改（UPDATE 命中数=1 实返）

## 范围外

- fin_* 其余表/其余 surface 行
- 读方语义（无消费者，本钉不改任何运行时行为）

## 参考

- T-0276 票面审计注记②留案一（`docs/tasks/T-0276-dcd枚举臂关闭.md`）
- SG 写入径先例：本票 SG `app_config` INSERT gen=373（docker exec o-postgres psql）

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
