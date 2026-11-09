# T-0508 CEO 自运感知·票B——层扫异常→诊断票自动进水口

- 状态：完成（ops_event 完成行 2026-11-09 在案——机械轨票翻）
  T-0507 层扫行产出面；T-0507 已并 master aa8d533ab 在案）
- 级别：P2 判断径接线——只接进水口，诊断/判词/根修票全走现行轨

## 范围

1. **层扫升件→诊断草案**：扫 T-0507 层扫升件行（seat=MECH
   note=层扫定性<类目>:<对象>）→`fin_ticket_draft` INSERT
   诊断型草案（kind=diagnosis；signature=「层扫:<类目>:<对象>」
   dedup——同对象同类目在飞/已诊不重复起）；
2. **物料化语义**：草案票面由层扫行载荷成形（类目/对象/首报
   时间/计数轨迹）——不编根因不编修法，诊断席的事；
3. **消行语义**：诊断票收口/作废落终态行时，对应层扫异常
   视为已诊——同 signature 后续升件按现行先例判据处理
   （复发≠新草案的语义照 dm_queue 排除律）；
4. 接线点=mech_tick 层扫窗尾段或 ticketdraft 径内——施工裁形，
   但**必须走 fin_ticket_draft 正径**不直写 docs/tasks。

## 判据

1. 人造层扫升件→fin_ticket_draft pending 行落库（signature
   dedup 生效：同对象重报零新草案）；
2. 草案物料化→诊断票经派单→裁定落案——人造异常全链零人触
   可复现；
3. 诊断票终态后同 signature 再报不炸不重复起（先例语义不破）；
4. 全谱零新增 FAIL/SKIP。

## 禁线

- 接线只产生草案行——根因定性、修法选择、根修票立件
  全归现行裁定径，本票不发明任何判断；
- 未知病形归宿=诊断席判断面——扫报矛盾存在即完成使命；
- 不碰钱径。

## 结果回填（施工——待验）

- 实现摘要：`internal/portfolio/hunger/layerdraft.go` 新件+
  `drafter.go` 编排改面——`DraftTickets` 拆双径：探针径改名
  `draftProbeTickets`（原逻辑逐字保留）+新增 `draftLayerScans`
  层扫径。层扫径=扫 `ops_event`（seat='MECH'∧event='升件'∧
  note LIKE '层扫定性%'∧24h 窗）→note 解析 fail-closed
  （「——」定界取 pair=<类目>:<对象>，首冒号拆类目/对象，
  残缺即弃）→signature 聚类（「层扫:<类目>:<对象>」票面钉死
  字面，pair 原文直嵌不改写）→`fin_ticket_draft` INSERT
  kind='diagnosis' pending+起草行（seat=决策席 ticket=T-0393
  同探针径账法）。物料化零新件——payload 契约同构
  ticketdraft 既有槽位（slug/title/evidence_ids/evidence_note/
  evidence_query/domain_hint），slug='layerscan-'+可读面+
  签名 crc6 尾（materialize 幂等 glob 按 slug 全域匹配，
  crc 尾防跨签名误并票号）。
- 接线点裁形：取「ticketdraft 径内」——drafter 起草拍=fin_
  ticket_draft 唯一产线（hungerTick 6h 拍与探针族同源）；
  消费面只读 ops_event 升件行契约不依赖 T-0507 码面合入态
  （施工期间 T-0507 已并 master aa8d533ab，契约对拍一致）。
  未取「mech_tick 层扫窗尾段」——那会把消费逻辑焊进生产件
  尾段且与票A 边界相交。
- dedup 语义钉（票面「在飞/已诊不重复起」落地）：
  ①pending 草案占位；②materialized 票 latest 事件∈
  {完成,作废}（drafterDedup 终态词集同源）=已诊→签名级
  永闭（复发≠新草案——票面消行语义即签名关账，非定时窗）；
  ③materialized 票非终态=在飞占位（含无事件链/挂起/审中）；
  ④同签名 24h 任意态草案=防灌窗（只管未诊签名 dismissed/
  merged 边缘面）。跳过行复用 drafterSkipRow 24h 前缀去重。
- 物料化载荷=层扫行载荷成形不编根因：Title=「层扫异常诊断票:
  <类目>@<对象>」，EvidenceNote=「首报 <全期 min(ts)>｜签名
  全期升件 N 行｜近24h窗 M 行｜<末行升件原文≤400字>」（聚合
  LIKE '——' 定界+无分隔畸形行 exact 兜底，LIKE 元字符转义），
  DomainHint 只指认「T-0507 层扫窗机械枚举不变量破例+根因/
  修法/根修票归现行裁定径」不预设病灶。
- 测试证据：`layerdraft_test.go` 五场景全绿——①人造升件→
  pending diagnosis 草案落库（signature/kind/lesion_key/载荷
  轨迹字段逐项断言）+起草行；②同签名再报零新草案+异签名
  另起；③materialized 在飞占位（dedup 跳过行「在飞票=T-9100」
  引证）→票翻完成后同签名再报仍零产（「已诊票=T-9100」跳过行
  引证——已诊永闭与探针径「终态释键」语义分野钉死）；④畸形
  note/伪 MECH 席/非升件事件全弃零草案；⑤双径零串味
  （观测盲探针径+层扫径同拍各一互不染）。存量
  drafter_test 五场景+hunger/lifecycle 整包回归全绿。
- check_all：scoped 定向谱全链绿（变更面=.go+docs——hunger/
  lifecycle/反向闭包+恒测册 20 包全 ok 零 FAIL/SKIP），戳
  `check-green-83f086632e88caf695b045d314978ef19be6778c`。
- 判据对照：①人造层扫升件→pending 行落库+signature dedup
  生效=桩测锚①②；②物料化→诊断票经派单→裁定落案走现行轨
  ——**范围外发现①**：`meta.autoapprove.registry` 册无
  「层扫:*」键（册语义=签名→钉定文件径白名单，诊断票工作面
  本无固定径）→层扫候选票将滞留等 DM/COS 批（2h 滞留升件
  呈 CEO 消化径在案）；「全链零人触」在现行批准径下需另裁
  批准面，本票范围=进水口不裁；③诊断票终态后同 signature
  再报不重复起=桩测锚③已诊永闭；④全谱零新增 FAIL/SKIP=
  check_all scoped 绿戳在案。
- 偏离说明：kind='diagnosis' 与 fin_ticket_draft
  pending_dedup 唯一索引（kind∈{rootcause_diag,
  trade_postmortem}）不相交——Go 级 dedup 为唯一闸（与探针
  类 kind 同例，该索引本不覆盖探针签名族，未扩面）；
  lesion_key=pair（<类目>:<对象>）与 signature 尾段同值
  自描述。
- 范围外发现（交付备注不自修）：
  ①机批册缺「层扫:*」批准径（上详判据②——若裁定要真零人触
  需另立批准面：册键语义不适配诊断票自由取证面，或裁
  「诊断票批准=DM/COS 常径」显式化）；
  ②「draft-滞留」类目喂诊断径的链式放大面——层扫
  draft-滞留:<tid> 升件→诊断票物料化→新候选票再滞留>1h
  →新签名再报，放大由签名唯一性+WIP 闸+批准积压限界，
  是否给该类目摘出诊断径归裁定席判断（本票按票面「扫报
  矛盾存在即完成使命」全类目照收未摘）；
  ③升件失领（T-0507 h面）认领语义与本票草案消费的关系——
  层扫升件行 ticket=原异常票号或空（系统件），诊断票为新
  票号，认领词事件不落原升件 ticket——物料化是否应补认领
  行属 T-0507/裁定径语义，本票未动。
- 坑处置：无新增坑（dedup 跳过行自身 24h 前缀去重致测试
  二次跳过行被压=先例语义非坑，桩测按语义清前记断言）。
