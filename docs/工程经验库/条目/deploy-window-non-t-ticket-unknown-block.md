---
schema_version: 2
id: deploy-window-non-t-ticket-unknown-block
kind: pitfall
risk: medium
status: verified
keywords: [deploy_window_due, ticket_block_faces, 非T前缀票, unknown, 永双挡, GW-001, 挡件谱系, 部署窗, 伪键票, 相位判据]
scope: [scripts/meta/deploy_window_due.sh, 部署窗机械轨, ops_event 票相位, GW-001 族, 伪键票]
summary: deploy_window_due 挡件判据按 T-NNNN 拼支式/工位——非T前缀票（GW-001/伪键族）num 剥离失效→saw=0→unknown 永双挡 img+oci，凡挂 pending/review 相位即永久压窗（实证 GW-001 鬼影挡 9 件 3.9h 靠 24h 兜底逃生）；修=支/树推导收窄 T-NNNN 规整形显式不可定+票面自证判据（票文件路径全非部署面→none 不挡/涉部署面按标挡/不可判→unknown 不退化）
first_seen: 2026-11-09
last_verified: 2026-11-09
skills: [施工席]
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0498-deploy窗挡件谱系-非T票unknown永挡根修.md（病灶实证段+结果回填先红双点）
  - type: doc
    ref: scripts/meta/deploy_window_due.sh（ticket_block_faces/ticket_doc_faces 判据②）
  - type: doc
    ref: scripts/meta/tests/deploy_window_faces_test.sh（桩⑪-⑮ 票面自证断言）
---

## 事实/症状

`deploy_window_due.sh` 判据②挡件谱系对 pending/review 相位票调
`ticket_block_faces`：`ticket_refs` 按 `t/NNNN*` 支式 + `/opt/o-wt/tNNNN`
工位两面找代码面，两皆空=`unknown`→`mark_blocked` 保守双挡 img+oci。
**非 T 前缀票结构性中招**：跟踪票/总票（GW-001）、伪键票
（DEPLOYNF/DOOR4/GATEFIX/PIPEFIX/BASESWP/DBP3a 族）无 T-NNNN 编号 →
`num=${1#T-}` 剥离失效 → 支式 `t/GW-001*` 永不存在 → `saw=0` →
`unknown` → **凡挂 pending/review 相位即永双挡**。

实证：GW-001 六段全 merge 后，各段收口移交行把它反复顶回 review
相位 → deploy_window state=有挡 rc=11 持续，批 9 件龄 3.9h 堆积，
全靠 ≥24h 兜底节拍逃生门放行（DEPLOY-WINDOW-OVERRUN 升件随发）。

## 根因或策略解释

「支/树皆不可定→保守双挡」的语义只对「可能有码而看不到」成立——
它隐含假设「票号=T-NNNN ⇒ 支式/工位可推导」。非 T 前缀票打破该
假设后，推导面不是「查无此支」而是「根本无从查」——两种不可定被
同一 `unknown` 桶装走，且无任何自证逃生径：票面自证零部署面的票
与「可能有码看不到」的票被同等对待，前者被误挡成永久死锁形。

## 处置

（T-0498 已落，桩⑪-⑮ 钉死）：

- 支/树推导收窄至 T-NNNN 规整形（≥4 位数字+可选字母后缀）——非 T
  前缀与异形 id=显式不可定，不误拼 `t/<id>*`/`/opt/o-wt/t<id>`；
- `saw=0` 新增票面自证判据 `ticket_doc`/`ticket_doc_faces`：票件定位
  docs/tasks[ /done]→docs/infra 三域首配，路径 token 归一仓根引用
  （`/opt/o[/wt/<树>]/`→`o/`）+首段顶层目录闸+根级部署件名单补抽，
  全非部署面→none 不挡/涉部署面→diff_block_faces 同谱按标挡/
  票件缺或零 token→unknown 保守双挡不退化；
- 通则（可迁移）：凡机械判据按票号推导支/树/文件的，相位面必须给
  「显式不可定」分支+自证逃生，禁默认拼接 T 式。

## 证据

- check: `bash scripts/meta/tests/deploy_window_faces_test.sh`
- result: ①-⑮ 全绿；先红双点——同断言对 master 版脚本跑⑪红
  （`待裁:T-0904(不可判→双挡) 待裁:GW-900(不可判→双挡)` rc=11），
  本枝⑪-⑮ PASS
- ref: T-0498 票面结果回填段

## 反证条件

- check: 若未来票号体系统一含非 T 前缀正本（如 GW-001 转 T 编号），
  或 ops_event ticket 键域加前缀约束闸
- falsifies_when: ticket_block_faces 支/树推导面被重写为票号无关
  的实态扫描（如直接扫 /opt/o-wt/* 与 origin/t/* 全量比对）——
  本条目「推导面」前提失效需重审

## 适用边界

- 只规 deploy_window_due.sh 判据②挡件谱系；dm_queue 相位投影、
  mech_tick 收敛窗执行径、deploy-faces.tsv 分类正本均不动；
- 票面自证按全文件路径 token 抽取（票面格式异构无统一涉及面解析
  钦定）——方向保守：散文提及部署路径也挡（误挡向安全侧），
  「无法判明→unknown」兜底不变；
- 实证对照残留：真仓 GW-001 票面散见 internal/cmd 部署路径→
  判 both 挡（自证非零部署面，属「误挡有正证可消」子集外，
  24h 兜底节拍仍其逃生门）——本修消的是 docs-only 自证子集。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（判据逻辑坑非量化阈值病灶——「永久误挡」由
  DEPLOY-WINDOW-OVERRUN 升件+deploy_window state=有挡 rc=11 观测面
  既有兜底承，无新增量表适配）
- class: n/a
- query: n/a
- threshold: n/a
