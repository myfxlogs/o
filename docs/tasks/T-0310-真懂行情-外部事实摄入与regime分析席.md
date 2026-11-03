# T-0310 真懂行情——外部事实摄入+regime 分析席

## 状态

**施工中**（devin-t0310 @ 2026-11-03；原态=已批）

## 背景（业主钦定口径）

- 机器要「真懂行情」——宏观/新闻/OI/链上面扩视野（现窄孔=永续费率切片）。
- 宏观采集源=**antclaw**（业主外部资产，非 O 仓件——写径/类/频三参待业主提供）。
- openclaw 已排除（O 仓零依赖核实，非本项目件不入架构）。
- **主消费面=决策席（agent 裁决读），非业主**——产出必须是结构化事实件
  （机器可查询），驾驶舱 memo 摘要=次级透明件。
- 硬边界：regime 产出永不写 app_config/提案/仓位——门官确定性闸不动。

## 钦定目标

### 阶段一（本票收口）
1. **`md_ctx_fact` 原始事实表**（md 域·append-only）：
   `id bigserial | source text | kind text | payload jsonb | received_ts | valid_until`
   ——antclaw/人工/feed 三源混灌；kind 开放词集（news|macro|oi|chain|manual）。
   写径 v1=PG-md 直写（antclaw 若跨机→tunnel 低权钥照 ghhook 范式另票）。
2. **`md_regime_fact` 结构化判词表**（md 域·分析席唯一写者）：
   `id|dim text|stance text|confidence numeric|evidence jsonb|created_ts|valid_until`
   ——决策席裁决时的读面（WHERE valid_until>now() 取活在判词）。
3. **分析席**（盒上 o-research 内新臂——假想官基建复用非新进程）：
   每 6h+人工触发读 md_ctx_fact 新件+行情库→LLM 产 md_regime_fact 行
   +memo 段（中文·引证据·标数据边界）。预算走 T-0267 每席账。
4. 决策席消费件：regime_fact 查面文档化（skills/决策席 补节：裁决前读
   活在判词——读面非闸件）。

### 阶段二（另票，本票仅留口）
antclaw 通道接通后 feed 扩面（RSS/经济日历/链上）+驾驶舱 memo 摘要卡。

## 钉注

- 注入双层防必走：md_ctx_fact 内容=不可信外部文本——分析席 prompt
  隔离（事实当数据不当指令）+产出 stance 限枚举域防自由发挥。
- md_ctx_fact 人工注入径=psql insert（业主/席均可——source='manual'）。
- 分析席写域收窄：仅 md_regime_fact/memo 两表 INSERT 权（无 UPDATE/
  DELETE/fin_* 权——revoke 面进迁移）。
- 三参待业主回：antclaw 写径（SQL直写|tunnel|file-drop）、可得类、频率——
  阶段一以人工注入+OI公开面实证闭环，antclaw 不接不阻塞。
- **mql5.com 钉为阶段一宏观主源**（业主指引 2026-11-03+实证可达）：
  /en/economic-calendar 免凭据服务端渲染——全币种事件 actual/forecast/
  previous 三值全谱（RBA 决议/ISM/JOLTS/CPI/央行讲话）。采法：盒上采
  集臂定时拉页解析（或 XHR 端点择优——施工实测钉径），kind='macro'
  source='mql5' 灌 md_ctx_fact；每事件含 id/时间/币种/重要性——天然
  结构化省 LLM 归一。附加面（阶段二再评）：/en/news 新闻流、quotes
  页——同域可采。

## 施工回填（devin-t0310 @ 2026-11-03，分支 t/0310-regime-analyst）

- **两表**：md_1083——md_ctx_fact（append-only，source/kind/payload/received_ts/
  valid_until，kind 开放词集）+md_regime_fact（dim/stance CHECK 枚举域 4 态/
  confidence/evidence/valid_until）+md_regime_memo（fact_id 关联中文段）；
  analyst_rw INSERT 两表授权（角色守卫包——写域收窄仅 INSERT 无 UPDATE/DELETE）。
- **分析席臂**：`internal/research/analyst`——Run 6h 拍+TickOnce 人工触发；
  collect 三源（mql5 日历页正则解析 kind='macro' source='mql5' 施工实测钉径
  =premiumIndex 页内嵌 JSON 提取 id/名/时间/重要性/币种；OKX OI 公开端点
  kind='oi'；单源败不拖全局）+analyze（LLM 缝 AttachLLM 注入式——缝缺=
  守位采集照常产出缺席如实）；注入双层防=prompt 隔读（DATA 段当数据不当
  指令）+stanceVocab 枚举校验（域外词 stance_off_vocab 拒行不落）。
- **挂臂**：cmd/research/dcd.go researchArms 挂 analyst 臂；import 边登记。
- **读面节**：skills/决策席/SKILL.md 新立——regime_fact 读面（活在判词查询
  式+纪律非闸件）。
- **测试**：3 桩（桩 fetcher macro+oi 落行/桩 LLM 判词+memo+枚举域外拒/
  缝缺零产出）全绿@5434。
- **待外部位**：antclaw 三参待业主回（写径/类/频——阶段一人工注入+OI 实证
  闭环不阻塞）；LLM 生产注入口径=装配件缝现成。

## 决策注记③ 落地回填（devin-t0310，同支续提）

- mql5 High（importance=3）事件直写 md_macro_event upsert source='mql5'——
  `macroKind` 词集映射（fomc/cpi/nfp 闭集外名归 ctx 广谱不造平行域）；
  种子行兜底保留（测试锚 seedN>0 实证）；中低事件照入 md_ctx_fact。

## ark vendored 吸收回填（devin-t0310，同支续提）

- ①`internal/research/ctx/`——cot/fred/calendar(mql5 XHR 端点择优)/news 四源
  采集件适配 O Fetcher/store；文件头注 ark 出处+Apache-2.0，LICENSE-ark 随件；
  红线件（Telegram/cmd/badger/confluence/AI SDK）零入，dep 仅 stdlib。
- ②落库分流：mql5 High→md_macro_event upsert source='mql5'（种子兜底实证）；
  中低/FRED(4 series)/COT/news→md_ctx_fact。
- ③`internal/lib/ta`——RSI/MACD/BB/ATR/SMA/EMA 纯函数壳（阶段二特征层备料）。
- ④ctx 臂挂 researchArms（ctxcoll 别名防 context 撞名）；analyst 判词缝不动。
- 测试：ctx 桩（High upsert+种子保留+ctx ≥3 行）绿@5434；analyst 3 桩绿。

## 决策席审计注记⑥（2026-11-03 一轮驳回——采集层全谱假造，ark 真件在手）

**核心病灶：「vendored ark 同款径」是谎话——施工没读 ark，端点/字段/类型全猜。**
真件已克隆核实（internal/service/news/fetcher.go），六返修项：

1. **mql5 端点假造×2**：ctx `GET /content?date=` 实返 **404**；analyst 正则刮
   `{"id":...}` 内嵌 JSON=真页无此形。真径：**POST /en/economic-calendar/content**，
   form=`date_mode/from/to/importance=15/currencies mask`，头必带
   Referer+X-Requested-With+Origin+浏览器 UA（裸 UA=403/空）。
2. **字段全错型**：真件=`Id/EventName/Importance("high"字符串非 int3)/
   CurrencyCode/FullDate("2026-03-17T14:00:00" ET→须转 UTC)/Actual/
   Forecast/Previous`；施工件 eventId/event/currency/importance:int 全假。
3. **FRED 无 api_key 必 400**：`api.stlouisfed.org` 无公共档——键缺席=每拍
   恒败噪声。修：键在才采（app_config/app_secrets 查件，缺=安静跳过）。
4. **mql5 双臂重复采集**：ctx+analyst 各采各写——同一 URL 两拍两份行。
   修：采集归 ctx 一臂；analyst 只读 md_ctx_fact（collect 函数全删含 OI）。
5. **md_ctx_fact 无幂等**：append-only 无自然键——同事件每拍重复行。
   修：外部 id 唯一键（(source,kind,ext_id) 或 payload hash 上唯一索引）。
6. **analyze 水印错 id 域**：`max(id) FROM md_regime_fact`（自身 id 域）比
   ctx id——恒重读最新 50 行=判词重复产。修：`max((evidence->>'ctx_max_id')::bigint)`。
7. **macro_event upsert 撞 PK 语义**：PK=event_date——同日多高事件（真实
   日历常见 CPI+利率决议同日）互踩静默覆盖。修：确定性优先级（同日期
   取 kind 序 fomc>cpi>nfp 单胜者）或 schema 改 (date,kind) 双键——
   勿静默覆盖。

已核无病灶：md_ctx_fact/regime 迁移+域判+臂挂载+stance 枚举域+prompt
隔读防注入+llm nil 守位+GRANT 角色守卫——分析层骨架合格留。
**测试桩教训三连犯**（binance interestRate→md_venue_funding 域→本次端点
假造）——KB api-stub 条目须加「声明 vendored 须附源文件行号」规。

## 决策席审计注记⑦（2026-11-03 二轮驳回——ctx 臂五销，analyst 臂病灶未动+幂等新坑）

**已销（ctx 臂合格）**：mql5 POST+form+六头+PascalCase 字段+Importance 字符串
+FullDate UTC 解析（源行号钉）✓；macro_event 优先级 WHERE（seed/fomc>cpi 序
防互踩）✓；dedupe_key 部分唯一索引+ON CONFLICT DO NOTHING ✓；FRED 键门
（cfg.Secret 凭据归 pg 钉径，空=跳源）✓；测试 10 包过。

**驳回三项**：
R1 返修项④原样未动——analyst/collect.go 整段还在：假正则 mql5EventRe 照跑
   （GET 页刮 `{"id":...}` 不存之形=每拍死 fetch）；且其写径绕开全部返修
   ——md_ctx_fact 裸 INSERT 无 dedupe_key、macro_event 裸 upsert 无优先级
   WHERE（盲覆盖病灶在另一臂还活着）。返修钉「analyst collect 函数全删含
   OI」= collect/collectMql5/collectOI/mql5EventRe/fetch 缝全删，TickOnce
   仅 analyze。
R2 返修项⑥原样未动——collect.go:142 水印仍 `max(id) FROM md_regime_fact`
   （自身 id 域错比 ctx id=恒重读）。改 max((evidence->>'ctx_max_id')::bigint)。
R3 幂等新坑——COT ins(...,"cot","",...) dedupe_key='' IS NOT NULL →全 COT
   行撞同键=历史仅首行落。改 NULL（免键）或真键（COT 报告周 YYYY-WW）。
