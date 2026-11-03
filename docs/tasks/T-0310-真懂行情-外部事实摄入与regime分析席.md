# T-0310 真懂行情——外部事实摄入+regime 分析席

## 状态

**已批（决策立案——业主令：产出不给业主给决策席；antclaw=宏观采集源）**

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


## 决策注记③（2026-11-03 —— 仓内考古新锚）

`md_macro_event` 第八桥表实存（fin_0067 SG 镜像+md_0020 正本）——
「宏观高冲击日历事实域」：32 行种子 FOMC/NFP/CPI（source='seed' 至
2027-10），funding eval `applyMacro` 已在读注 evidence.macro。
**antclaw 全仓无实体**——env 键幽灵+meta 隧道注册名，采集从未落地。

钉注修订：
1. **mql5 High 事件直写 md_macro_event**（source='mql5' 替种子——桥
   镜像/eval 消费链全现成，勿造平行日历域）；中低事件入 md_ctx_fact。
2. md_ctx_fact 收窄为「广谱上下文」域（news/oi/chain/antclaw/manual）——
   与 md_macro_event（钦定高冲击事件面）分工不重叠。
3. 种子行处置：mql5 首采后对重叠事件 source 改 'mql5' upsert；种子行
   保留为兜底（远端缺席不删——事件存在性以 seed+live 并集为准）。
4. 业主待供缩为一件：antclaw 写径（接不接都不阻塞）。


## 决策注记④（2026-11-03 —— ark-intelligent 原型钦定）

业主定：采集原型=github.com/myfxlogs/ark-intelligent（myfxlogs=业主自家
域仓——Telegram 部已拆不用）。**采集臂照 ark internal/ 移植非从零写**：
mql5 日历/FRED/COT 三客户端端点+解析+限速现成可抄；价格链/Bybit 微观
不重复采；TA/信号/AI 层不移植（产信号非喂事实——与分析席职责异构）。
FRED_API_KEY 免费件——业主提供时入 app_secrets 轨（env 域 O_FRED_API_KEY
模式已在 check 册）。采集臂住 o-research（scout Fetcher 范式同款目录
internal/research/ctx/ 新包）。

阶段二候选（业主质询后细化）：ark TA 指标数学→分析席特征层（波动率/
动量入 regime 判词——喂事实非产信号）；ark 回测方法论（per-regime hit
rate/校准）→评史官策略评审升级件；ark AI 层 prompt/限速当参考读。方向性
信号引擎永不移植（投机径=已拆病灶）。


## 决策注记⑤（2026-11-03 —— 业主令「吸收 ark 入 O」）

「移植」升级为「吸收」：ark 采集件 vendored 进 O 仓（Apache-2.0——
vendor 件须随 LICENSE 署名段）。吸收边界钦定：
- **收**：internal/{cot,fred,calendar(mql5),news,price} 采集客户端 + pkg
  指示数学（TA 指标库——阶段二特征层备料）
- **不收**：Telegram 适配/cmd bot 主环/BadgerDB 存储适配/信号 confluence
  引擎/AI 层（职责异构+投机径病灶）
- **落地径**：internal/research/ctx/ 新包——ark 客户端代码适配 O 的
  Fetcher/store 模式（scout 同款），头部注 ark 出处+Apache-2.0；指标
  数学放 internal/lib/ta/（本票建好壳+关键指标，特征层接入属阶段二）。
- **依赖纪律**：ark 第三方 dep 只带采集必需件（重试/限速器），
  Gemini/Claude SDK 不入——分析席复用假想官既有 LLM 径。
