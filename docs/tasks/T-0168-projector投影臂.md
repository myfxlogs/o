# T-0168 cmd/projector 投影臂——活跃集谓词七镜像表+fin_proposal 双向（D-0030 §三/§四）

## 态：审计通过
审计席：headless @ 2026-10-30
## 立项：决策席 @ 2026-10-26
## 领取：zcode-glm 施工席 @ 2026-10-27（支 t/0168-projector-arm，基 40772bc——T-0167 已合 ff684df）
## 决策档：D-0030

## 施工
1. `cmd/projector`：LISTEN PG-md 本地 ev_*（零轮询）→活跃集谓词查最新行→
   写 SG 镜像表（ON CONFLICT DO UPDATE WHERE ts>EXCLUDED.ts——per-table
   键照 domain.tsv；md_macro_event 无 ts 用 event_date 键语义，D-0032）。
2. 活跃集来源：SG 侧 `fin_active_set`（fin_0069 已建名——fin_position
   closed_at IS NULL ∪ fin_strategy 非 retired 触发器维护，ev_active_set
   NOTIFY）——projector 订阅帧刷谓词；**启动/重连全表 reconciler**（D-0032：
   谓词自算 fin_position/fin_strategy 不赖种子——同径兼治 0069 无回填）。
3. **八桥表**镜像谓词照 D-0030 §三+domain.tsv bridge 册（md_macro_event
   第八桥含）。
4. fin_proposal 双向：PG-md→SG 投影全表；SG state/retire_reason/retired_at
   变更（**ev_proposal 双侧帧**——dev insert 与 SG 状态组更新各发，D-0032
   §三裁）→回写 PG-md；SG 属三列禁投影覆盖（唯一双向面——桩重点）。
5. projector_silent 告警（镜像 max(ts) 超阈）+ roster 注册。
6. 失效语义（D-0030 §七）：projector 死→SG 镜像 ts 陈→sizing/MTM 拒新动作
   fail-closed；既有仓平仓=SG 本地 MT5 feed+OKX 私有 WS 不赖镜像。

## 桩锚
- 活跃集外 inst 行→零投影；活跃集内→投影；SG 改 proposal state→PG-md 回写。
- projector 停→SG 镜像 ts 陈→sizing 拒开仓（T-0128 闸复用）实证。
## 验收锚
- 隧道载荷 ≤300 行/分（实拍）；镜像表 ts 新鲜 ≤60s；双向回写零丢；
  reconciler 重启自愈实证（杀连→全表对账→集齐）。

## 态：待验
## 结果回填（施工席 zcode-glm @ 2026-10-27，支 t/0168-projector-arm）

### 逐项施工

- **① cmd/projector**（internal/projector 包+薄 main）：双 DSN（MD_PG_DSN 本机
  PG-md/SG_PG_DSN 隧道）；md 链 boot 幂等迁移（帧源自保障）；岗终 fail-closed
  （单败即 Fatal——systemd 重启→reconciler 自愈径）。零轮询：events.ListenLoopChan
  （本票加 channel 感知变体——wake 携 channel 名，塌陷语义不变）分派八桥拍。
- **② 活跃集**：谓词自算 SG `fin_position closed_at IS NULL ∪ fin_strategy
  非 retired`（不赖 fin_active_set 种子——0069 无回填同径兼治，D-0032 裁）；
  ev_active_set 帧触发自算重刷。**reconciler**：boot 全量（逐 id 集差+ts 水印
  归零重拉——同 ts 删行只有归零可见）+10min 活性兜底（max-id 窗+now-10min
  重刷片——events.go 期界豁免类，sched-exempt 留痕）。
- **③ 八桥谓词**（照 D-0030 §三）：md_ticks/md_mark_index/md_funding=活跃集；
  md_signal=carry_diff/xvenue_spread 全量+funding_view/depth5 活跃集键；
  md_dcd_product=fin_strategy dcd_call 活策略 inst_id（=product_id，码面实勘
  paper.go:151/174 同径）；fin_opp_eval=仅被 fin_proposal 引用行；fin_proposal
  全表；md_macro_event 全表 diff。upsert per-table 键照 §六表：ts 判六表
  WHERE EXCLUDED.ts>本表.ts；md_funding 同 ts DO NOTHING；macro event_date
  全列覆写；**前向提案禁写 state 列组**（回环构造性断绝——SG ev_proposal_sg
  按 UPDATE OF state 触发）。numeric 文本搬运保精度。桥表集对 domain.tsv 册
  fail-closed（migrations.BridgeTables embed API——消费面与册单源）。
- **④ ev_proposal 双侧帧**：md_1067（PG-md AFTER INSERT 正向源）+fin_0070
  （SG AFTER UPDATE OF state 回写源）+md_markidx/md_signal/md_dcd 帧补全
  （0028 缺项——投影零轮询前置）。回写=SG 活跃/近期退役行对拍 md 三列差
  UPDATE（repairProposalStates）；eval 先提案后落（SG eval_id FK 序）。
- **⑤ projector_silent+席位**：30s beat（roster「投影官」bridge 席）+镜像
  max(ts) 陈阈 2min 转沿告警（Alerter log 面——feishu 凭据归 SG，dev 侧
  零解密钥不装）；部署件 deploy/o-projector.service（用户级，Slice=o-prod.slice）。
- **⑥ 失效语义**：复用 T-0128 60s 陈闸（sizing.LatestPx）——桩锚实证（下）。
- **boot 守卫（票面外推定，报备）**：a) SG ev_proposal_sg_trg 在位探针
  （fin_0070 未轮换即拒起——聋哑 fail-closed）；b) **跨库 id 键空间切序守卫**
  （md fin_proposal/fin_opp_eval 序列 next ≤ SG legacy max(id) 即拒——SG 63.5万
  legacy id vs PG-md 新序列从 1 起，前向投影会覆写 legacy 行=实害；
  T-0169 setval 前置机检化）。

### 桩锚（O_TEST_PG_DSN 5434 全绿）

- 活跃集外零投影/集内投影（ticks/mark_index/signal/dcd 四谓词形+ETH 集外断言）
- 双向：正投→SG 改态→md 收敛（armed）；退役三列回写；对账重跑 state 列组
  各持正本（前向 SQL 无 state 列组——构造性+同 id 重投影产线不存在实证注记）
- 杀连自愈：镜像行清+断窗新提案→全量对账集齐且不越活跃集谓词
- macro 全列覆写 diff；跨库切序守卫红/绿两相（setval 径同形）
- 镜像陈→sizing.LatestPx 60s 闸拒（D-0030 §七失效语义实证）

### 机检

build✓ vet✓ gofmt✓ check_migrations 七断言✓ capcheck✓ check_env_reads
（MD_PG_DSN 入册——报备）✓ check_import_direction（projector 三边入册——报备）✓
check_no_forbidden_patterns（Ticker 两处 sched-exempt-ok 心跳/活性类留痕）✓
check_rejects_keys✓ check-lines✓ 全仓回归见收口行。

### 验收锚对账

- 隧道载荷 ≤300 行/分（实拍）：**未达（前置缺席）**——实拍须 PG-md 有真实
  写流（collector/research DSN 切换=T-0169）；本票交付机制+桩锚，实拍移交
  T-0169 对拍期首拍。
- 镜像表 ts 新鲜 ≤60s：同上移交（桩面已证投影延迟=查询+批写量级）。
- 双向回写零丢：PASS（桩锚——armed/retired 三列+对账重跑归属断言）。
- reconciler 重启自愈实证（杀连→全表对账→集齐）：PASS（桩锚 TestReconcileSelfHeal）。

### 偏离/报备（交决策裁）

1. **T-0169 前置清单**（projector 可起的三缺）：a) SG o-core 轮换（fin_0070
   应用）；b) SG 侧 projector_rw 角色授予（八桥镜像 upsert+fin_position/
   fin_strategy 读+registry beat+app_config 读——现 collector_rw 面不足）；
   c) md 序列 setval 抬过 SG legacy max（守卫报错文案即操作指引）。
2. md_markidx/md_signal/md_dcd 三帧=票面「零轮询」的必要推定（0028 缺项，
   md_1067 一并落——超单风险低，报备）。
3. events.ListenLoopChan=ListenLoop 加法变体（原 API 零改，塌陷语义不变——
   塌陷合流期 channel 名只保末到，余者水印/对账径兜，注记在案）。
4. env 白名单 MD_PG_DSN、import 方向表 projector 三边——两表头注均「新增须
   评审」，本席 D-520 份内先决，决策复核。
5. 投影官告警走 log 面（feishu 凭据 SG 侧密文——dev 零 master key 不装），
   projector 死亡面由 roster last_beat 陈化覆盖（SG 侧哨）。

## 验收（决策席 2026-10-27 @merge 后落）

**判：PASS——合入 master。**

独立取证：SG 侧六键约束逐一对拍全在（ticks/venue,inst_id·markidx/venue,
inst_id,channel·signal/kind,key·dcd/venue,product_id·eval id·proposal id·
macro event_date·funding UQ 待 fin_0066 轮换应用）；check_migrations 七断言/
build×4/vet 绿；projector 测试 6 桩全过（集外零投影/双向三列回写/杀连自愈/
macro 覆写/切序守卫两相/60s 闸拒）。

**要害裁**：①跨库 id 键空间守卫=施工捕真坑（md 序列 1 起 vs SG 63.5万
legacy——裸投即覆写），拒起哨+T-0169 前置清单三件全收；②**本席补一坑**：
fin_opp_eval 自然键 UNIQUE 与按 id upsert 的毒丸环——setval 不覆盖，已钉
T-0169 前置（首选 COPY legacy 行入 PG-md）；③载荷/新鲜两锚前置缺席如实
移交 T-0169 对拍期——票据未声称实测，不收「达标」只收「机制+桩锚」；
④三帧补全/ListenLoopChan 加法变体/env+import 两册=份内先决合规，收。

票态：✅ 验收合入。T-0169 候派前置：本票机制毕+T-0167 PG-md 在跑——
剩 SG 轮换窗与三缺清单（见 T-0169 承接段）。

## 判决注记（审计席 headless @ 2026-10-30——收口 merge 87dfc6a 逐锚对账+产线实拍）

- 锚①隧道载荷 ≤300 行/分（实拍）：**字面未达——判锚数值失准、报管理席裁决锚口径**（依规不自改锚）。实拍：SG `pg_stat_user_tables` 60s 增量桥面 upd 合计 ≈3,400/min（md_mark_index ~2,140 + fin_opp_eval ~975 + md_ticks ~272 + md_signal ~61；proposal/funding/macro/dcd = 0）。serve 谓词收窄产线生效实证：近 2min 新鲜 inst=107 ≡ fin_active_set 107，集外仅 SG 本地共写 BTC-USDT 一键——投影独写下限 markidx 单表 ~2,100/min（107 inst × 3s 节流拍率），任何符票面谓词语义的正确实现在现活跃集规模下均破 300（锚源 D-0030 ≈215 行/分预估，未含「活跃集数 × 源拍率」下限）。保护意旨已由字节口径实证达成：SG→隧道向 ~18KB/s（45s ss 混合向采样）vs 切前基线 498KB/s ≈96% 降，与 T-0169 钦定同法实拍 31.6KB/s（≥80% 锚过）同量级；SG CPU 降证见 T-0169 承接段。
- 锚②镜像表 ts 新鲜 ≤60s：**过**——SG 实拍（~10:20Z）md_ticks 0.2s / md_mark_index 1.0s / md_signal 3.7s / fin_opp_eval 1.3s。md_dcd_product max ts=02:41Z 系源侧 catalog 静默非投影滞后（10min 活性对账+事件双径在役，源有新行必至——eval 引用面 30min 内 424 行续投为佐）；md_funding ts=资金时点语义（未来值正态）；proposal 无 ts 列 / macro event_date 键语义票定。
- 锚③双向回写零丢：**过**——桩面 TestProposalBidirectional 绿（审计席 worktree 复跑 internal/projector 15/15）；产证：SG ev_proposal_sg_trg 在册、2090 行 fin_retired 态变发帧面在，前代 md 正本（dev:5433，OCI 迁移前）实拍 fin_armed/fin_retired+armed_ttl_expired+retired_at 三列回写落 md；repairProposalStates=diff 写非事件回放，丢帧由 boot 对账+10min 活性兜底收敛=构造性零丢。
- 锚④reconciler 重启自愈实证：**过**——桩面 TestReconcileSelfHeal 绿（杀连→全表对账→集齐且不越活跃集谓词）；产证=dev journal 10-27/28 崩环窗每次重启皆达「对账毕——活跃集 NNN inst」（~1750+ 次全数收敛），终态 282 inst/139 品；现 OCI 在役 beat 9.7s 鲜（投影官 roster 实证），历 OCI 迁移+serve 终切多次重启镜像仍 <4s 鲜。
- 机制面抽核（87dfc6a diff）：八桥 upsert 键照 D-0030 §六表全对（ts 判六表 EXCLUDED.ts>ts / funding 同 ts DO NOTHING / macro event_date 全列覆写）；uProposal 前向无 state 列组→SG ev_proposal_sg UPDATE OF state 不触=回环构造性断绝；eval 先于提案落（FK 序）在码；projector_silent 哨 mirrorStaleSince 2min 阈+roster 30s beat 在册；boot 守卫含 fin_0070 探针+跨库 id 键空间切序守卫（产证真拦：journal 10-27 08:25 next=max 拒起）。
- 注记存案（不阻断判决）：①md_dcd_product 源侧静默 7.6h 与 T-0243/T-0244 feed 病灶同域，归该票链不属本票投影径；②本席行口径系 pg_stat 共享授权面（projector_rw/collector_rw/research_rw/o 均持写权）——归因偏差单向虚高（本地共写混入 ≤~250/min），投影独写下限仍 7× 超阈，不改变锚①失准判定；③dualwrite→serve 终切后实测形态与票内 dualwrite 期回填口径不同——产线现态以本注记为准。
- 复验注（审计席 headless 重飞 @ 2026-10-30——87dfc6a diff 独立对账+产线重采样，四锚维持通过）：②镜像 ts 复实拍 md_ticks 0.3s/markidx 1.1s/signal 1.9s/eval 30s（funding ts=资金时点未来值不计滞）；③SG ev_proposal_sg_trg 在册、fin_retired 2092/armed 2 行、前代 md(dev:5433) retire 三列落行 399——回写产证维持；④投影官 roster active、beat 26s 鲜（OCI 在役）；①45s pg_stat 重采样桥面写 2,036 行≈2,715/min——同量级复证锚①字面失准（活跃集×源拍率下限即破 300，意旨字节降已证），续报管理席裁决锚口径。**新见偏离一处**（不阻断）：收口 diff 内 roster 仅 Beat 未 Register——册行缺席则 beat 岗终环，票面「roster 注册」项收口时点未达；T-0169（12ed90cb）补 Register 幂等后投产无碍，投产序列本就押 T-0169 前置清单故实害零，备录防再犯。
