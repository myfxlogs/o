---
id: hypothesis-chain-release-boundary
title: 自生假设→提案→paper 放行边界——六段逐段审计现态（工单#10693 姊妹档）
kind: design
date: 2026-11-12
source: 工单#10606（业主令④D件——目标发起放行 system 核链+实证+canon 载边界）
---

# 自生假设/提案链放行边界（2026-11-12 产线实谱审计）

钦定盈利目的内「自生假设/提案经 eval→paper→live 免疫链照旧为闸」
的现行链各段放行面（产线实查非推断）：

| 段 | 放行机制 | 闸性 |
|---|---|---|
| gapscan→假想官→卡 | ev_gap 事件臂自驱（hypothesis/run.go） | 自治 |
| 卡批准（decide） | `tools/hypo decide` 窄门——会话认知闸（外席=O 的思考，非人手）；骨检 adopted 须 execfam 在册 | 认知闸（设计内，卡是需求非执行件） |
| 注册（adopted→模板+评估臂） | 骨架五件交付=要码修（新族 executor 本质须码面落成） | 构建闸（设计内） |
| 评估→fin_proposal | evaluator 枚举臂（funding-hedge/basis/dcd/carry/vol-sell 全自驱） | 自治 |
| 提案→paper | admitProposals 自动开户（唯一闸=execfam 册+同键去重复用） | 自治 |
| paper→live | gradgate/TTL 迁移机（proposed→paper→probe→live→retired 闭集） | 自治 |

**「不经人手」判读**：会话=O 的认知层（D-0045「你就是机器在思考」），
会话 decide=O 自决非人手；真阻自主的两闸皆为设计内——新族卡收养须
骨架码面（executor 不能自写），卡裁决须认知在场（防无人橡皮章）。

**产线实证行**（自生提案走通 eval→paper→live 不经人手）：
- `fin_strategy #3544` funding-hedge：proposal#3745→paper(10-25)
  →probe(10-31)→live(10-31)→retired(11-10) 全迁自动；
- `#287108` perp_quarter_spread/#287109 quarter_basis_long：
  proposal#413263/413262→paper(11-06)→probe(11-08)；
- `dcd_put` 批 2874xx：evaluator 自生提案→paper 11-09 连批。

**现存缺口（如实非病灶）**：#172286 trend_follow 卡在 proposed——
trend_follow 未入 execfam 册（8 族闭集），adopted 骨检必拒；要放行
该族=骨架五件要码修，非链断。
