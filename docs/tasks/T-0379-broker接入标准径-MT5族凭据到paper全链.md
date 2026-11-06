# T-0379 broker 接入标准径——MT5 族凭据到 paper 全链

- 状态：完成（2026-11-06 决策席裁决通过——收口支
  t/0379-mt5-broker-onboard@bdf774a3 合 master）
- 领取：devin-o-worker-t0379 @ 2026-11-06（常径机派施工席；提示词
  docs/prompts/T-0379-O-Dev施工提示词.md）
- 优先级：P0 旁支——venue 转化首案 T-0377 是单点，本件把
  「新 MT5 broker」变标准径（觅食面 venue 维扩容）
- 收口移交归：决策席

## 病灶

venue 链前半在（scout→candidate→recommended→注册请求推），
**凭据落地后断尾**：fin_venue_candidate 到 registered 无后续；
探测/映射/激活每步都靠 bespoke 票（MT5 首案=T-0377 全票施工）。
业主令边界=「注册/凭证/KYC 归业主,其余全归机器」——接入段
必须机件化。

## 范围（MT5 族通用——同 mtapi 协议零 per-venue 码）

1. `internal/venue/onboard`（或按导入方向册挂合法位）接入拍：
   发现新凭证（readcred 域 `<broker>.mt5.*` 键形，命名约
   定票面钉死）→ 四拍流水：
   ①探测：mtapi Connect→AccountSummary→SymbolParams（能力
     账=skills/mt5-* 五件，照 canon 不猜）→ 写 onboard
     report（ops_event「venue接入」行+evidence 锚）；
   ②符号映射草拟：broker 符号表×OKX inst 匹配——产出 map
     草案（不落闸,产出物=proposal 件）；
   ③map 落闸：默认业主单次批（条件请求推 O 群携草案——
     `bridge.xvenue.map` owner 类键不破域）；备选附注=
     `bridge.xvenue.map.<login>` 子键降 dm 类+硬校验（符号
     必须在 broker SymbolParams 自报表+swap 参数可核）——
     降域另票议,本件不擅自降；
   ④激活：map 落闸后 xvenue-carry 模板对该 broker 符号开
     paper 评估（复用 T-0377 admitXvenuePair 径,零新模板）。
2. 状态机补尾：fin_venue_candidate.state 增补
   `probed/mapped/paper`（迁移件——registered 之后三态）；
3. 多账户面：mt5 事件已 login 粒度——台账/观测面按
   `<venue>:<login>` 键,venue_fail_gate 同粒度；
4. 凭据缺席/败径：fail-closed+ops_event 自报（T-0369 律）。

## 验收锚

① 桩测全链：假凭证→探测→草案→map 闸→paper 行——
   四拍每拍 ops_event 落行；
② 真实户 EXR 3001157 当「首个 broker」干跑①②（探测+草案
   不激活——已有 map 在库不重复落闸）实证协议面；
③ 状态机迁移+查询面断言；
④ 凭据坏/网关不可达/符号表空三败径桩测全 fail-closed+自报；
⑤ check_all 绿戳对口；镜像面件并入发窗。

## 红线

- 不动凭证域写径（owner 类物理不可达照旧）；
- 不动真单径——本件止于 paper 激活,live 晋升照生命周期
  既有判据+live_switch；
- 不降 bridge.* 键域（备选仅附注）；
- crypto CEX 接入不包——per-venue 票另立。**CEX skills 缺口
  附注**（业主 2026-11-06）：ant 上游无 CEX skill（仅 mt-*）
  ——结构性因：MT 族共享 mtapi 单协议一账通用，CEX 无共享
  协议每户 bespoke API。定径：此后每立一张 CEX venue 接入票，
  其验收锚必含「skills/<venue>-* 能力账入 canon」（照
  skills/okx-* 七件范式——OKX 是 CEX 侧唯一已证模板），
  venue 转化一张票沉淀一件能力账，不空过。

## 结果回填

**实现摘要**（MT5 族 broker-agnostic——mtapi 单协议零 per-venue 码）

- 新臂 `internal/venue/onboard`（挂 o-research researchArms——PG-md 正本
  写 fin_venue_candidate、fin 池写 ops_event/condreq、md 副本读 app_config；
  finPool LISTEN config_changed（app_secrets 凭据落/app_config map 落同帧脊）
  +10min 活性兜底，事件驱动零轮询）。
- 凭据发现：app_secrets `<broker>.mt5.<field>`（票面钉死键形；单例 `mt5.*`
  =「首个 broker」venue='mt5'）族枚举+cryptoutil.Open 正径解密——凭据域
  只读，明文禁出盘/禁日志。
- 四拍：①探测=mtapi Dial(gateway_addr)→Connect（server 径优先/host 多候选）
  →AccountSummary→SymbolList→SymbolParams（能力账 canon 序）→ops_event
  「venue接入」+evidence.onboard 锚→'probed'；②映射草拟=broker 符号表
  ×okx.symbols 剥后缀比基（USDTM/USDM/USDT/USDC/USD/PERP 长先剥）→草案
  proposal 件 evidence 锚+condreq kind='permission' surface='bridge.xvenue.map'
  携草案推 O 群→'mapped'；③map 闸=bridge.xvenue.map.<login> 子键优先→
  平铺 bridge.xvenue.map，落值须 ∈ broker 探测符号自报表（备选注硬校验
  同义——越表 fail-closed 自报不激活）；④激活→'paper'+ops_event——paper
  开户走既有 admitXvenuePair 提案有机径（零新模板零新径）。
- 败径 fail-closed+自报：凭据坏/网关不可达/符号表空=stay 'registered'
  +ops_event esc_class='接入败' esc_scope='<venue>:<login>'+evidence
  attempts/last_attempt_at（30min 重试 backoff）。
- 多账户粒度：ops_event esc_scope/evidence 键=<venue>:<login>（venue 列
  CHECK 禁冒号——组合键不落 venue 列）。
- 状态机迁移：md_1090+fin_0118 同语句 ALTER 补 CHECK
  probed/mapped/paper（桥表 DDL 同形断言⑥对口，master 水位 1089+1/117+1）。
- 干跑件 `tools/mtonboard`（-venue/-beats 截拍——EXR 首案 beats=2 探测+
  草案不激活）。

**测试证据**（O_TEST_PG_DSN 桩测四锚全绿）

- TestOnboardFullChain：假凭证 Seal 入 app_secrets→scan→registered→probed
  →mapped（草案对 BTC-USDT-SWAP→BTCUSDm 断言+condreq pending 行断言）
  →写 bridge.xvenue.map→scan→'paper'；ops_event「venue接入」4 行
  beat=探测/草案/map闸/paper激活 逐拍断言；esc_scope='fake1:3001157'
  <venue>:<login> 粒度断言；非法态 CHECK 拒断言（锚①③）。
- TestOnboardBadCreds/GatewayDown/EmptySymbols：三败径各 stay
  'registered'+esc_class='接入败' 自报行断言（锚④）。
- TestCredKeySplit：键形拆分锚（单例/前缀/杂形拒）。
- go build+vet 绿；check_migrations 七断言过（链内唯一/序号/桥表同形）。

**偏离说明**

- 锚② EXR 干跑未在本 session 执行：dev 盒无 OCI PG-md 可达径
  （fin_venue_candidate 正本侧不可达——5544 隧道只到 SG fin；工具双池
  面就绪）。runbook：merge+o-research 部署后臂自然跑 EXR（mt5.* 凭据
  在册=自动发现→探测→草案→map 已在库免重复落闸→mapped；闸检测
  bridge.xvenue.map 命中→paper），或 `tools/mtonboard -venue mt5 -beats 2`
  手动截拍实证①②。
- ④激活解释：paper 开户不直写 fin_strategy（research_rw 无授权+提案
  有机径即票面钦定复用径）——'paper' 态=激活声明，开户随提案链。
- map 多 broker 并存=平铺 bridge.xvenue.map 单 broker 表达限——
  bridge.xvenue.map.<login> 子键闸检测已在位，降 dm 类另票（票面钦定）。

[施工收口:T-0379] @<SHA待落>
