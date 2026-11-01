# T-0288 OKX DCD 期权族生死定性（T-0118 拆解·族票③）

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 立项 | `[角色:决策]` @ 2026-11-01（T-0118 母票按机会族拆解首批） |
| 领取 | devin-施工席 @ 2026-11-01 |
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

## 结果回填（devin-施工席 2026-11-01）

**产出账**：7d eval 16.6k（dcd_call 8677=5625 rejected+2523 unevaluable+529 proposed；dcd_put 7973=5721 rejected+2252 unevaluable）；proposed 529 全数 10-25 爆批——此后零 proposed=**产出面已死非旺**。净 APR 分布 rejected 中位 0.27~0.66 高值但 verdict 全拒=评分口径与拒绝口径断链。

**退役边界台账**：md_dcd_product 22231 行但 max(ts)=10-30 02:41——枚举臂 T-0276 退役即断供，评估面在陈数据上空转（10-30 后 eval 件=陈数据噪声，白烧算力）。capability_surface okx/dcd=f+模板双禁+臂退役三面齐锁。

**executor 面**：DCD 执行席（T-0160 投产）在册但 fin_strategy dcd_call 全 retired 244；venue_untradeable 51001 假合约族积累=果非因在案。

**生死裁备**：**选 B——评估面同退**。依据：数据供面已断（md_dcd_product 30h 陈化且无人喂），16.6k 后续 eval=陈数据空转噪声；候选池语义失效（陈价评的「机会」无意义）。回启条件清单（若将来选 C）：①md_dcd_product 供面复起（枚举臂或替代馈送）②capability_surface okx/dcd=t ③模板启封+venue_untradeable 假合约清理三件套齐——缺一不启。

**范围外确认**：enabled/capability 零翻、枚举臂零重启、执行席零触。

**审计通过**——管理席审计 2026-11-01：复核过——选B（评估面同退）判据成立：md_dcd_product max(ts)=10-30断供实证=枚举臂退役后评估在陈数据空转（16.6k后续eval=噪声定性准）；proposed 529全10-25爆批此后绝迹与全局面一致；回启三件套清单可执行。
