# 研究进水口规格——外部策略谱→假设矿（集量清单件,趁隙施工）

> 业主令 2026-11-10：重心转生成半；开源策略谱不止 Hummingbot，
> 眼界放开——普查成谱,常态监视。

## 分工

| 谁 | 干什么 |
|---|---|
| 值守（业主会话） | 本规格起草+修订 |
| 研究臂会话 | 按本档执行普查/监视,产 hypothesis 行 |
| 机器管道 | eval→paper→probe→live 照旧接管,无新件 |
| 唤知/审计 | 不管——健康域非研究域 |

## 源阶梯

```
一阶 分类学正本: Hummingbot 策略目录(market_making/arb/
  spot_perp/grid 谱)——定族骨架用,不深读实现
二阶 策略知识富矿(各家独货,防单源偏):
  QuantConnect Lean 社区算法库——期权/期限/统计套利型
  vn.py 生态——CTA/期现/跨期族(中文量化独档)
  freqtrade/Jesse/Backtrader 社区库——指标驱动族
  利基仓——funding-arb/basis-trade/跨所搬运专仓
三阶 venue 产品册(非开源但必扫——钱实际在的地方):
  OKX earn/借贷/结构化票/期限合约
  MT5 经纪商 swap/展期表(在册 venue 先行)
```

## 三道筛（假设行入场门）

```
① 约束筛: 本金~$16k/零售API/无速度优势——
   延迟敏感/做市/高频型直接毙不入评估
② EV算式筛: 必须载可算收益式（费率×结算窗×磨损×资金占比），
   无式=不成立
③ 证伪锚筛: 写明「什么证据出现=此假设亏」——无锚=半成品
```

## 产出格式（fin_opp_hypothesis 行）

```
venue+family+leg_primitives+requirements+evidence:
  evidence={source:'<仓/册>',ev_formula:'<式>',falsify:'<锚>',
            constraint_fit:'<过哪道筛/卡哪道>'}
```

## 节奏

```
首件=一次普查: 建《策略普查表》族×源×约束×EV草式
  （成果落 docs/drafts/→评审→谱入库）
常态=监视拍: venue 新品/源库新策略→新增假设行
  （触发档写入研究 runbook:③矿进水触发）
```

## 不做

```
✗ 不抄代码进仓——进来的是假设不是实现
✗ 不评比 bot 平台功能——评的是策略形状对约束
✗ 不动 eval 阈/生命周期闸——新假设照样被拒单
```
