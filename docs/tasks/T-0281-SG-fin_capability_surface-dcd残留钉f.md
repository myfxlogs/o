# T-0281 SG fin_capability_surface okx/dcd 残留钉 f（T-0276 审计留案①）

| 字段 | 值 |
|---|---|
| 状态 | 待验（claude-kimi-t8 收口 @2026-11-01）|
| 立项 | `[角色:决策]` @ 2026-10-31（MANAGE-3414③ 裁决——审计注记②留案一「是否顺带钉」裁=钉，单列小票走流程） |
| 领取 | claude-kimi-t8 @ 2026-11-01 |
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


## 结果回填（claude-kimi-t8 @ 2026-11-01）

**实做**：SG `o-postgres` 单行 UPDATE（带 `AND enabled=true` 幂等闸）：
`UPDATE fin_capability_surface SET enabled=false WHERE venue='okx' AND
surface='dcd' AND enabled=true` → **实返 UPDATE 1**；走 docker exec
o-postgres psql 径（票面参考先例 gen=373 同径）。

分支: t/0281-capresid（已推送=已交付）
验收锚对账:
  - 锚1 SG 实查=f: PASS（UPDATE 后 SELECT 实返 okx|dcd|f；40 分钟+多轮
    投影周期后复拍仍 f）
  - 锚2 过副本同步周期复拍双侧 f 无回弹: PASS*（SG 侧复拍 f+全表
    checksum 钉前后一致 676db2610edd1c7d51307820b0dd8196 零漂移；PG-md
    侧=f 据审计在案实查——本席无 OCI 直达径〔vps ssh 不达〕，未亲证，
    如实标注；另勘：fin_capability_surface 无 projector 周期同步臂
    〔mirror/replica 零行写〕，SG 残 t 溯源=md_1044 种子非同步滞后——
    回弹面仅新环境重放，见发现③）
  - 锚3 dcd-test 行与其余行零改+命中数=1: PASS（UPDATE 1 实返；
    dcd-test=f 钉前后不变；全表 checksum 钉前后一致=其余行零动）
机检: build✓ vet✓ gofmt✓ env✓ rejects✓ migrations✓ bridge✓ test✓（check_all
  全绿——本票零码改，docs-only 面）
自检 A–I: A 单行 UPDATE 无新径；B 直击歧离；C 零码零渣；D 幂等闸
  （enabled=true 谓词）保重跑安全；E 范围内零越；F 票面同步；G 如上；
  H SG 实拍双轮+checksum；I 非展示面
涉及面: SG o-postgres fin_capability_surface 单行（库面）+本票面
范围外确认: fin_* 其余表/其余 surface 行零触；本地 dev 基线库零触（发现②
  交决策裁）

**施工发现（交决策裁，未越单处理）**：
1. 本地表**无周期同步臂**（mirror/replica 零写）——「副本同步周期」前提
   不成立，回弹面=新环境 migrate 重放再种 t（md_1044 INSERT enabled=true
   ON CONFLICT DO NOTHING，新建库即 t）；另案候选=种子行改 f 或迁移尾补钉。
2. 本地 dev 基线库（本机 o-postgres）okx/dcd 同残 t——票面涉及面=SG 单
   面，未动；同病灶他副本面是否钉，交决策。
3. PG-md 亲证缺径：本席 ssh 面（sg/vps）无 OCI 通道，PG-md=f 引 T-0276/
   本票立项审计实查在案。
移交: 决策席
[施工收口:T-0281] @70b801b
