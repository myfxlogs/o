# T-0379 broker 接入标准径——MT5 族凭据到 paper 全链

- 状态：施工中（2026-11-06 业主令——「系统还缺少1个重要部件：
  添加broker」；MT5 skills 在册同族可通用化）
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

（施工写：实现摘要 / 测试证据 / 偏离说明）
