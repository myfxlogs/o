# T-0385 教训矿工拍——事件账自动挖工程案 pending

- 状态：完成（2026-11-06 决策席裁决通过——收口 3715ef9c 合 master 4e3e648e）
- 领取：devin-o-worker-t0385 @ 2026-11-06（常径机派施工席；提示词
  docs/prompts/T-0385-O-Dev施工提示词.md；wt=/opt/o-wt/t0385
  支 t/0385-lesson-miner，bootstrap 降级径起步——rsync gen/dist 快照）
- 优先级：P1——自我迭代第⑤段（记忆）补盲
- 收口移交归：决策席

## 病灶

KB 沉淀靠人想起来写——T-0164 式「注释自白盲区」证明没人
写就不沉淀。ops_event 账里升件/驳回/异常/返修签名每天都在
产，没有矿工拍=高信号教训随账沉没。

## 范围

1. `internal/portfolio/hunger` 同槽或同正径矿工拍（慢拍
   建议 12h——记忆面非急诊）：
   扫 ops_event 窗（升件/驳回/异常/返修/收口偏离）聚类
   → 签名级去重（esc_class+scope/reject key/票号维度）→
   达聚类阈（默认 ≥2 同类或单件高危）起草**工程教训
   pending 行**；
2. 落表：复用 fin_econ_lesson pending→verdict 机件（扩
   kind='eng' 或新 fin_eng_lesson——工人按面谱择,迁移号
   水位+1）；
3. 裁定径：pending 行入裁定席既有无头裁流（裁定席 SKILL
   裁域扩 eng 案—— verdict 行照旧三态）；
4. 起草载荷：签名/证据锚（ops_event id 列）/定性草案/
   建议条目名/**验证态四档**（auto_verified/ai_evaluated/
   human_confirmed/unverified——博主指南四态标签：无实证
   锚的草案落 unverified 显态，防「没验」混同「验过」）——
   机读五件,裁定席按此裁；
5. 防灌闸：日额 ≤5 草案+同窗签名去重+「无高信号」时零产
   （benny 沉默律）。

## 验收锚

① 桩测三形：聚类达阈=pending 行落（签名/锚/草案齐）/
   未达阈=零行 / 同签名重扫=去重不产；
② 裁定席 eng 案裁流桩测：pending→verdict→KB 条目落；
③ 实况窗断言：近期事件账（0380 ff-merge/0382 种键案）
   矿工应产出对应 pending（回放实证）；
④ 日额闸断言（>5 截断）；
⑤ check_all 绿戳对口；镜像面件并入发窗。

## 红线

- 只挖不裁不修——裁定席判词照旧,矿工零 KB 直写；
- 不引入外部服务——纯 PG 账内聚类；
- 签名去重防灌水；
- 不动真单径/业主面键。

## 结果回填

- **实现摘要**：
  - 迁移 `fin_0120_eng_lesson.sql`——工程案锚表独立成表（面谱择：
    econ 载荷=econv1 protojson 契约面 vs eng 载荷=签名/证据锚/草案/
    条目名/验证态机读五件——异谱共表=两案互染，故不挤 kind CHECK
    集择新表；迁移号水位 fin_0119→0120+登记+域册 fin_eng_lesson=fin）。
    列面：signature 聚类签名/cluster_n/verify_state 四档 CHECK/
    numbers jsonb/status pending|verdicted|canon/verdict_ref；
    roster_rw SELECT+UPDATE（裁定席隧道凭据）+o_fin_ro SELECT 补读。
  - `internal/portfolio/hunger/miner.go`——MineLessons 矿工拍：
    ops_event 12h 窗扫（签名集=升件/驳回/异常/返修＋裁决 note 驳回*
    双写件＋收口 note 含「偏离」——生产真谱对拍：14d 升件31（带类7）/
    异常36/驳回16/返修1）→签名聚类（升件=esc_class+esc_scope 等值键；其余=
    类+票号/席别键）→达阈起稿（同窗同签名≥2 或单件高危=升件/驳回/
    异常/红线类域——0380 ff-merge 单件升件实案须产故升件入高危集；
    返修/收口偏离=例程形签名仍走≥2 阈防双起稿）→签名级去重
    （fin_eng_lesson 同签名 pending 恒挡/任意态 7d 冷却）→日额闸
    ≤5+无高信号零产。pending 载荷五件齐（签名列/evidence_ids=
    ops_event id 锚列/draft 定性/suggested_entry kebab 条名/
    verify_state=ai_evaluated——机器聚类稿带实证锚非 unverified）；
    同拍 ops_event `教训矿工` 留账行（自主≠无账，行不入签名集零自指）。
  - `internal/portfolio/lifecycle/miner.go`——minerLoop 独立 ticker
    12h（同正径不入 6h 饥饿槽——票面钦慢拍）+拉起即首挖（T-0381
    首侦同例）+败情升件自报行（写败/LIFECYCLE-MINER 24h 同窗去重）。
  - 裁定席裁域扩 eng 案：`scripts/meta/verdict_tick.sh` 水印闸改
    econ+eng 两表 pending 合>0（eng 表缺=查询败兜底算零不炸；
    DEVIN_BIN 桩测覆写位——生产钦定径不动）；`skills/裁定席/
    SKILL.md` 裁域扩——eng 读面/词集（kb_entry:<slug>→工程经验库
    条目+kbgen gen；kb_stage:<slug>→待归纳 staging；dismiss 只
    verdict_ref）/落面三步+verdicted 收尾同径。
  - `docs/数据流.md` 补矿工拍链路段（链路变更票同 commit 更新规）。
- **测试证据**：
  - 锚①桩测三形齐：`go test ./internal/portfolio/hunger -run Miner`
    10/10 PASS——达阈 pending 落（签名/锚/草案/条目名/验证态五件断言）/
    未达阈零行/同签名重扫去重不产/高危单件三件产/窗外行不入/沉默律零产。
  - 锚③实况窗回放：0380 ff-merge（ops_event#2350 真行形状复刻——
    COS 无类升件单件）→升件:T-0380 pending 产；0382 种键案
    （#2390 裁决驳回+#2391 驳回双写真行复刻）→驳回:T-0382 cluster_n=2
    pending 产（TestMinerReplay0380ffMerge/TestMinerReplay0382SeedKey）。
  - 锚④日额闸：6 簇截断=5（TestMinerDailyCapTruncates——返修中
    发现并根修日额双计 bug：已落库行随查询自见勿再计 produced）。
  - 接线桩测：`go test ./internal/portfolio/lifecycle -run Miner`
    2/2 PASS——minerLoop 拉起即首挖+逐拍连调+ctx 净退/假败自报行
    +24h 去重。
  - 锚②裁定席 eng 裁流：`bash scripts/meta/tests/verdict_tick_test.sh`
    PASS 四案（双零不烧/eng-only 烧/表缺兜底/econ 照旧）——pending→
    verdict→KB 条目落段=devin 无头 run 件（SKILL runbook 已扩域，
    全链实证须部署后真 run 回放，此边界如实记）。
  - `go build ./...` + `go vet` + `gofmt` 净。
- **偏离说明**：
  - bootstrap 降级径起步（rsync gen/dist 快照——本票无 proto 面动，
    快照与支面同宗无漂）。
  - 实况窗回放=桩测面真行形状复刻（真库回放须 fin_0120 部署后——
    不合并不部署令下此为可达边界）；裁定席 eng 裁全链实证同边界。
  - 镜像面件（Go 码+迁移）——进阻塞集，发窗归决策席判据。
