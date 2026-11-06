# T-0405 枚举面补 deferred 两族+面清单快照 diff

- 状态：已批（2026-11-06 业主令「立吧」——残余三审收口：
  提案重复查证非缺陷/表读写对恰为 0394 deferred ③族/
  该建未建类对面清单 diff 化）
- 领取：已施工（待验——收口移交决策席）
- 优先级：P1——发现层收口件（0394 枚举面族补齐）
- 收口移交归：决策席

## 实证

- `internal/portfolio/hunger/depenum.go:8`——「③表读写对
  /④配置对 deferred（票面钦可选臂——family 槽预留）」；
- 「paper 策略活而 fin_paper_fill 零行」正落③族该罩位；
- 「该建未建」类残余——探针查产消对不查「系统该有什么」，
  对面清单快照+diff 退化为可见差集行。

## 范围

1. depenum ③族启用：**表写者↔读者对枚举**——枚举径同
   既有族（读面注册/命名规约，枚举器空集=fail-closed
   升件）；首谱至少含：fin_strategy(paper)→fin_paper_fill
   /ops_event 写者→消费者/fin_opp_eval→fin_proposal
   三对钉定（白名单槽照旧）；
2. depenum ④族启用：配置对（app_config 写者↔读者——
   键域谱面枚举）；
3. **面清单快照 diff**：每拍落面清单行（服务/表/产消对/
   模板/闸键域——机器可列面）与上拍 diff，新面=升件行
   （24h 去重同式）——快照存 ops 面窄表不建重型件；
4. 桩测：③族纸 paper 零 fill 检出/合法未读白名单豁/
   面清单新增行升件三态。

## 验收锚

① ③族枚举器生产谱实跑产出非空对清单（含 paper→fill）；
② ④族同式；③ 面清单 diff 新面升件桩测过；
④ check_all 绿戳对口。

## 红线

- 枚举器空集 fail-closed 律守（0394 钦定）；
- 只报不修——探针零写业务面；
- 白名单豁面照旧机制，禁枚举器误报洪水（dedup 24h）。

## 结果回填

**实现摘要**（diff ⊆ 票面三面）：

- ③族启用（depenum.go/depblind.go）：`depTablePairs` 钉定谱
  （读面注册——sig+产/消窗内行数 SQL 同列，$1=窗天数 7d）。
  票面钦三对：`table:fin_strategy[paper]→fin_paper_fill`
  （paper 策略在册逾窗应产记账而 fill 窗内零行=失明；
  entered_at<窗免新生 paper 误报）/
  `table:ops_event→ops_event[chain]`（写者活而链写覆写
  hash 窗内零行=链消费全死；部分断归 ops_chain_verify 裸行）/
  `table:fin_opp_eval→fin_proposal`（评估有产提案零=链失明）。
- ④族启用（depenum.go/depblind.go）：`configDomains` 枚举
  app_config 键域首段×`depConfigReaders` 读面册（域源=
  config-write-domain.tsv 全谱+种键域）——册外域=
  `config:<域>→∅` orphan 升件（死键显形）；册内域=常对
  留账恒不盲。
- 面清单快照 diff（facesnap.go+fin_0124）：每拍机器可列面
  五类（service=活跃席/table=public 基表/dep_pair=对签名/
  template=评估∪模板族/gate_domain=键域）落批
  ops_face_snapshot（batch=seq 派号）→上拍批差集=新面→
  「面清单新面」行+观测盲升件（esc_scope=face:<kind>:<face>，
  24h 去重同式）；首拍基线零升件；快照仅留最近两拍批
  （窄表自清非重型件）。挂 ProbeDeps 同拍槽（败情并入拍级错）。

**测试证据**：

- 桩测四件新增：③族 paper 零 fill 检出（含新生 paper 误报闸+
  腿对隔离）/③族白名单豁（豁免行携理由键）/④族死键 orphan
  检出/面清单三态（基线零升→新面升件→谱稳零新增+批≤2）；
- 旧锚适配：零误报锚补提案消费种（eval→proposal 新对不冤枉）；
  空集锚语义重钉（③④启用后枚举地板=钉定对+键域对，全谱空
  结构性不可达——len(pairs)==0 闸仍守，锚改断言两族入列）；
- `check_all` 全链绿（build/vet/import/gofmt/env/rejects/
  config-write-domain/migrations 七断言/bridge-audit/ops-chain/
  go test 全谱）——戳落 worktree check-green。

**偏离说明**：

- ops_event 消费者取「链写覆写覆面」（hash 非空窗内行数）——
  读侧不可机读，链覆写=消费者存活唯一可观测代理（部分断由
  ops_chain_verify 裸行检出，本对罩全灭形）；
- 「服务」面取 ops_event 近窗活跃席投影（席自报面——进程级
  服务清单库内不可得）；
- 升件行票号：面清单 diff 行携 T-0405（depblind 三族行仍携
  T-0394——升件面按立件票分域）。
