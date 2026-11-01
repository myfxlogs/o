# T-0288 OKX DCD 期权族生死定性（T-0118 拆解·族票③）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-11-01（T-0118 母票按机会族拆解首批） |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | 高（产出最旺 vs 全链禁用——最大错配族） |
| 涉及面 | `fin_opp_template(dcd_call/dcd_put)`、`fin_opp_eval`、`fin_capability_surface(okx/dcd)`、md_dcd_product、T-0276 退役账 |
| 执行径 | 无头/交互皆可 |

## 背景

最大错配族：eval 产出全系统最旺（dcd_call 8677+dcd_put 7973=7d
16.6k 件）但 ①模板双禁 ②capability_surface okx/dcd=f ③枚举臂
T-0276 建制退役（research.dcd.enabled=false+健康面摘牌）。
「看见最多 vs 全都做不了」显形族——生死定性=本族到底保留评估
面（候选池价值）还是评估面也收（算力租金白烧）。

## 钦定界

- 产出账：16.6k eval 的评分分布——高分件占比/拟 armed 命中面
  （评估在产机会还是产噪声）。
- 退役链回追：T-0276 枚举臂退役范围 vs 本票评估面——臂退役后
  eval 为何仍在产（谁喂数据、是否该同退）。
- executor 面：DCD 执行席（T-0160 投产件）现状——在册/故障/
  venue_untradeable 假合约族积累影响。
- 生死裁备：A 评估面续跑留候选池 B 评估面同退（禁 eval 出件）
  C 模板/能力面回启条件清单——三选一定性+依据。

## 验收标准

- [ ] eval 评分分布+高分占比实证（噪声/机会明判）
- [ ] 退役边界台账（枚举臂退 vs 评估面在产的接缝在案）
- [ ] executor/venue 假合约面实证
- [ ] A/B/C 生死定性+启用条件清单（若 C）

## 范围外/禁线

- 不翻转 enabled/capability 不重启枚举臂不动执行席
