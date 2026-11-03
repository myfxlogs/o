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
