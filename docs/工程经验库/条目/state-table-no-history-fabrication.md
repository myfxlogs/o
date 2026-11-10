---
schema_version: 2
id: state-table-no-history-fabrication
kind: pitfall
risk: high
status: verified
keywords: [状态表, 无史, 滑点, APR史, unmeasured_buffer, fin_slip_sample, 事实律, 虚构历史]
scope: [lifecycle, execution, eval, calibration, 数据面]
summary: 判词需「历史」而源为状态表（upsert 末帧无史）时，按惯例/印象虚构序列=违例；
  正解=自产留史+标注锚+延评三选一，禁虚构填数
first_seen: 2026-11-10
last_seen: 2026-11-10
evidence_refs:
  - type: ticket
    ref: T-0521
  - type: commit
    ref: 59deaf8a5
---

## 状态表无史：判词要「历史」时的虚构诱惑与三合法动作

- **实证**：T-0521 FinRL 镜照施工中两次直面此坑——①滑点校准想要
  历史滑点，但 md_ticks 是状态表（PK=venue+inst 只留最新帧），历史
  mark 不可回溯；②vol 入场判词想要 APR 30d 分位史，但 md_dcd_product
  与 fin_opp_eval 都是状态表（UNIQUE inst 键 upsert 末帧），两侧皆无史。
  诱惑=「按行业惯例编一段历史/中位数」填数——这正是事实律禁面。
- **三合法动作（法典化口径=工程纪律§事实律历史缺位谱）**：
  ①**自产留史**——每事件点现锁现记落 append-only 样本表，往后即真史
  （fin_slip_sample：成交腿落库即锁当时 mark+slip+mark_ts；鲜度闸
  ts-mark_ts<120s 防「旧 mark 冒当时价」）。攒够最小样本前③延评。
  ②**标注锚**——样本不足期估计值 detail 显式 unmeasured_buffer/
  default 非测量（frictionBps src 字段=审计可辨测/兜），实测足自动取代。
  ③**延评**——样本薄测出的率噪声>信号，fail-closed 不宣判不虚构基准。
- **参照系替代法**：需「相对基准」而无史时，先找**已存在的真实锚**
  再谈留史——vol 判词用提案自记 net_apr（立项时该产品被判定值得卖
  的真实价）作基准，「现价塌于立项半值=IV 穷不卖」因果链比「低于
  自史中位」更直接且零新基建。
- **关联病灶**：「最小样本闸」（frictionMinSamples=3/评估史 n≥5）是
  谱的一部分——薄样本宁可不覆写旧值；实测覆盖率缺口本身也是事实
  （样本行 mark NULL 落账=覆盖缺口显形而非虚构补齐）。
