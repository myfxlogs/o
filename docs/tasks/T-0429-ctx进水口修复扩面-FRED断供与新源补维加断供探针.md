# T-0429 ctx 进水口修复扩面——FRED 断供+新源补维+断供探针

- 状态：施工中
- 领取：外派施工（O-Dev 径，不占 DM 内联派工队列）；t0429@2026-11-07
- 优先级：P1——regime 判词六维饥荒实证，外部事实面
  实质停摆（判词自己报：仅 OKX OI 一管在滴水）
- 收口移交归：决策席
- 关联：T-0310/D-0043（regime 分析席立项——判词产出臂）、
  skills/ctx-sources（四源端点账+FRED v1/v2 界桩+UA 禁令
  **施工前必读**——FRED 断供大概率 Akamai 拉黑族病）；
  regime→消费面接线（另案挂后——水源先通）

## 病灶（11-07 regime 判词自证）

- FRED 断供：止 id976，零升件静默死；
- equity/fx/macro/commodity 四维零覆盖——判词连续拍
  「零新件承上拍」；
- 结构性缺：源死零探针——断供不升件=产出无人消费的
  同族病（数据源层版本）。

## 范围

1. **FRED 断供根修**：查死因（Akamai UA 拉黑族/端点
   变更/凭据期）——照 skills/ctx-sources 界桩修，
   禁自造 UA 重犯定案；
2. **新源补维**：按进水口枚举补 equity/fx/macro/
   commodity 至少各一可用源——源选以「免鉴权公开
   端点优先+既有采集臂同构」，新源端点账同票写回
   skills/ctx-sources（用中验错必修律）；
3. **断供探针**：每源陈旧阈升件（ops_event——
   阈按源节律定，如 FRED 日级源 48h 无新件=死）；
   探针消费归裁定席既有径，禁写孤儿 sink；
   **钦定样板=T-0410 `feedstale.go`**（日历 feed
   死闸同构件——探针形态/scope 判例域/阈钦定
   全照它，禁自创新形）；
4. **写域收窄不动**：采集件仍只写 md_* 域，
   零 fin_* 触。

## 验收

- 实机锚：FRED 新件 id>976 落库+至少两维原零覆盖
  维度有新件（部署后验）；
- 断供桩：模拟源死→阈到升件行可读（断言）；
- skills/ctx-sources 条目回写（新源端点账+死因案）；
- 绿戳对口+全链。

## 结果回填

- 状态：待验（外派径收口——回 DM 裁决；不合并不部署，实机锚部署后验）

### 死因勘查定案（2026-11-07 实机三重实证）

1. **FRED「断供」主病灶=结构自残非外因**：dedupe_key=序列名
   （md_1084）——首拍成功（id973-976，11-03 13:42）后任何成功拉取
   恒 `ON CONFLICT DO NOTHING` 零新行，成功也断供且零日志，与
   transport 死同象不可辨。**键/端点/transport/凭据今日 OCI 实测全
   通**（Go 默认 UA 400 业务响应×3、curl 200、KEY_LEN 32+200 实证
   ——凭据期/端点变更/Akamai 拉黑三候选全排除）；11-03 13:40
   `stream INTERNAL_ERROR` 一次=边池间歇踢流（fred-akamai 条目族，
   仅初始诱因）。
2. **ffcal 零行史死=windows-1252**：FF 周历 XML 头声明
   `windows-1252`（OCI curl 实拉 40KB 实证），Go xml 无 CharsetReader
   直接拒读→parse_err 全静默分叉（collectCalendar 原零日志）。
3. cftc（周更 11-06 鲜）/okx OI（日戳行在滴）本就活——病灶仅 FRED/ffcal
   两源+「源死零探针」观测盲。

### 实现摘要

- **FRED 根修**：dedupe_key=序列:观察日（新观察日=新键=新行，分析席
  id 水印增量可见）；扩列 SP500/NASDAQCOM（equity）+DCOILWTI/DHHNGSP
  （commodity）——stooq 机房 IP 全 404/Yahoo 429 实证后死端弃，走既有
  FRED 臂扩列（可用性实证优先，见 skill 取舍段）；逐列 fail-soft
  （原首列败 return 拖杀余列=断供扩面形）；kind 跟序列族
  （macro/equity/commodity——分析席 dim 自述）。
- **ffcal 根修**：Decoder.CharsetReader=charmap.Windows1252（实见声明
  位族显放，未声明族显错不瞎猜）+拉取/解析败 log 灯行（T-0311 返修⑤
  同形）。
- **新源 frankfurter**（fx 补维）：ECB 免鉴权公共面
  `api.frankfurter.dev/v1/latest?base=USD&symbols=EUR,JPY,GBP,CNY`，
  独立上游（FRED 断供不连坐 fx 维）；dedupe=USD:率日；kind=fx。
- **断供探针**（probe.go，feedstale.go 钦定形）：五源 row-age 巡检
  →ops_event `feed断更` 行+观测盲升件（esc_scope=CTX-<源>，判类册
  追记）；阈按源节律（FRED/frankfurter 96h——营业日周末缺口 72h，
  票面 48h 例每周末假报故上浮；ffcal 10d/cftc 9d/oi 48h）；零行史=
  断供极端形同报；24h 同窗去重；挂 TickOnce 尾槽（采集后验鲜）；
  败=log 自报非 fatal（errc=岗终径）；升件写径=AttachEscPool(finPool)
  ——ops_event 驻 fin 域，FIN role=review_rw INSERT 实证（research
  节点 md 池无 ops_event 面）。
- **零迁移零 schema 变更**：md_ctx_fact 开放 kind 词集+dedupe 键=数据
  面运行时形；分析席读形不变（同表同列同 payload 家族）。

### 测试证据（先红后绿）

- 红桩（修前实跑）：TestFFCharset1252 红「windows-1252 周历 High 事件
  未落 md_macro_event（charset 拒读形）: no rows」；TestFredDedupeNewObsDate
  红「dedupe_key 应含观察日（DGS10:2026-11-05）: n=0」。
- 绿验：ctx 包全测试过（charset/dedupe 转绿+fx 幂等+探针五形：陈行
  升件/鲜行静默/零行史同报/24h 去重/escPool 缺守位）+cmd/research 包
  过；TestCollectStub 扩面锚（kind fx/equity/commodity 各≥1+幂等保持）。
- 机检：check_all 绿戳（见收口行）。

### 验收锚对账

- 实机锚（FRED 新件 id>976+两维新件）：**待部署后验**——本票不合并不
  部署（外派收口纪律）；预测依据=键/端点全通+dedupe 键含观察日（部署
  后首拍即应落 8 列新键行）。
- 断供桩（模拟源死→升件行可读）：PASS——probe_test.go 五形断言全绿
  （TestProbeStaleLesionOnStaleRow：fred 行龄 100h>阈 96h→feed断更+
  观测盲/CTX-FRED 行落账可读）。
- skills/ctx-sources 回写：PASS——五源端点账+equity/commodity 取舍案+
  探针阈账+死因案全落（另 KB 两条新条目+kbgen 重建+数据流正本补 ctx
  臂段+判类册 CTX 行追记）。
- 绿戳对口+全链：PASS——check_all 全绿（戳号见收口）。

### 偏离说明

- 阈 96h vs 票面 48h 例：票面自载「阈按源节律定，如 FRED…48h」——
  「如」为例示非钦值；营业日源周末缺口 72h，48h 恒假报（alert 疲劳
  反噬探针公信），取 96h（周一缺报周二晚即报）。判据已在 skill 阈账
  显锚。
- equity/commodity 未走新免鉴权端点：stooq/Yahoo 对 OCI 机房 IP 实测
  全拒（404/429），「免鉴权公开端点优先」在可用性实证前让位——fx 维
  已由 frankfurter（免鉴权新源）满足新源补维本义。
- md_macro_event ffcal upsert 幂等形（源='seed' 兜底+优先级）未动——
  原样保持，仅复活其进水。
