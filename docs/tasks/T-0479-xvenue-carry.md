# T-0479 探针升件自动起草：观测盲@xvenue-carry

- 状态：已批（CTO 批 2026-11-08——病灶产核实：SG 直查 carry_diff 7d=2 行信号活、
  fin_opp_eval template ILIKE %xvenue% 7d=0 行、fin_opp_eval 全族 7d=140
  评估机本身在跑、fin_strategy 非 retired xvenue-carry=0——评估链消费面
  对该族零观测盲坐实）
- 领取：无头施工席常径（swe-2-flash-high）
- 优先级：P2（DM 维持起草档——非堵点族，部署前置缺角在 T-0481 域）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:xvenue-carry`，草案锚 fin_ticket_draft#27）

## 病灶（升件行证据引）

病灶签名：`观测盲:xvenue-carry`（kind=探针类 `观测盲`，key=病灶键 `xvenue-carry`）

末行升件原文：
> 族沉默 xvenue-carry：源信号 carry_diff 近7d 鲜行=2 而 fin_opp_eval 该族行=0——信号活而评估零观测盲区（探针只报不修，修归常票/饥饿响应）

证据引（ops_event 行 id）：#3226

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='xvenue-carry' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（CTO 批填——派单依赖闸对拍域）

1. `internal/research/hypothesis/`（run.go xvenue_carry 模板已注册——
   评估环为何零消费 carry_diff：调度位/threshold 闸/data_needs 取数面
   逐段核）；`internal/research/sigdist/xvenue.go`（carry_diff 产出行
   谱形与消费期望对拍）；`internal/projector/evalmap.go`（eval 映射
   消费面）；
2. 病灶根因定位与根修（签名面病灶本体——不超签名外衍）；
3. 桩测钉死断言（病灶存在断言修前红+同断言修后绿——T-0388 修复类先红桩律）；
4. 生产谱面零误报审计。

## 验收锚（骨架——DM 裁时补足）

① 桩测断言全绿（红/绿双点证据落结果回填）；
② 探针同窗不再报同签名病灶（修复后探针行随停）；
③ check_all 绿戳对口；并入发窗批。

## 红线（继承模板——不改不删）

- 只修本票签名病灶——不超单施工、不碰范围外文件；
- 不部署、不并 master——收口回决策席常径裁；
- 探针只报不修——本票修病灶本体，禁顺手改探针阈/签名面/类册；
- 钱径/凭据域/真单径语义禁碰。

## 结果回填

### 挂起诊断（2026-11-08 无头施工席——await=裁量：修域裁定）

**票面前提证伪**：评估环并非零消费——xvenue-carry 评估行在 PG-md
（OCI 5435 正本）持续产出且新鲜（2026-11-08 10:24 实测 rejected/
net_ann_economics——MT5 swap 成本腿 ~833-852bps 吃光 funding
365-565bps，判词正确）。o-research journal 同步实证评估臂逐拍在跑。

**真病灶=观测域错位（结构性，非码 bug）**：
- projector `-mode=serve` 在役——`qEvalFresh`（mirror.go:39）eval
  镜像谓词=**仅被 fin_proposal 引用行**（D-0030 §六写权矩阵钦定，
  lifecycle admit join 需要 ~20 行）；PG-md fin_opp_eval 实测
  ~93 upd/sec（n_tup_upd 60s 增量 5561）——全量镜像物理不可行，
  谓词是承重墙非可松件。
- xvenue-carry 评估全 rejected→零提案→SG 镜像恒零（SG 侧末行
  2026-10-30，系 dualwrite 时代/提案路径化石）。
- 族沉默探针（hunger Sensor→lifecycle.hungerTick，o-core@SG）与
  gauges.tsv `xvenue_carry_eval_zero`/`stake_carry_eval_zero` 同读
  SG fin_opp_eval——rejected-only 族结构性恒零→日频误报在案
  （ops_event #3226/#2450、#3446/#3445、#2869/#2868）。
- 探针族覆盖本身即化石依赖：SG fin_opp_eval 模板集缺 stake-carry
  （PG-md 有）——无化石行的族根本不入探针域。

**修域三径（皆出本票 diff 面，呈裁量）**：
①探针/量表测量面改判（hunger/silence.go+gauges.tsv——「eval 因
  谓词不可见」与「eval 链死」分域，代价=rejected-only 族 eval 死
  亦失明）；
②新增轻量 eval 活动脉冲桥（projector 派生 per-template max(ts)
  →SG 小表，探针/量表改读——新表=migration+桥册+canon，架构案级）；
③接受噪声+判词归档（族经济学本不成立，观测盲如实=无机会非断供）。

**域内核实零改动**：hypothesis/xcarry.go 消费闸（8h 源节拍界
T-0383 后正确）、sigdist/xvenue.go 产出谱形（direction 钉+双腿
年化 meta 全）、evalmap.go 提案译码面——三段逐项对拍无病灶。
