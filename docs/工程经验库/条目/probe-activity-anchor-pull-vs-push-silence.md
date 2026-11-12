---
schema_version: 2
id: probe-activity-anchor-pull-vs-push-silence
kind: pitfall
risk: medium
status: verified
keywords: [CALENDAR-FEED, 活性锚, 拉取侧, 推送侧, 事件驱动feed, 合法静默, 断更假阳, md_calendar_pull, md_calendar_push, T-0032]
scope: [事件驱动型 feed 活性探针, 采集臂 vs 产出表锚定口径]
summary: 事件驱动 feed 的断更探针若以「推送侧产出表末行龄」为活性锚，上游合法静默窗（无事件可推）=结构性假阳；活性锚必须钉「拉取侧动作产物」（采集臂自己留的行/戳），产出静默≠链路死。产线实证=CALENDAR-FEED 24.1h 界缘假阳→修口径换锚。
first_seen: 2026-11-11
last_verified: 2026-11-12
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT id,note FROM ops_event WHERE esc_scope='CALENDAR-FEED' OR esc_scope='face:table:md_calendar_pull'
    observed_at: 2026-11-12T00:30:00Z
  - type: commit
    ref: 5e47e34d（fix/cal-feed-caliber 探针口径修——拉取活性锚换推件产出锚）
  - type: commit
    ref: 11e23589（facesnap 补册 face:table:md_calendar_pull）
---

## 事实/症状

CALENDAR-FEED 探针以 `md_calendar_push`（高冲击事件推送落行表）末行
sent_at 龄>24h 判 feed 断更。升件#10654 龄=24.1h 恰越阈——但日历面
语义=「有高冲击事件才推」（T-0032 钦定无事件窗静默合法），探针自标
「两可如实」仍触发升件：界缘假阳占处置格。直接核 push 表还受权限
阻挡，处置只能靠政策判词推断。

## 根因或策略解释

锚错了侧。事件驱动面=上游按有无可推决定是否产出——**产出静默与
链路死亡在该语义下不可分**，拿产出侧当活性锚=把合法空态判成断供。
可分信号在拉取侧：采集臂每拍自己发起的拉取/解析动作落行
（md_calendar_pull）——臂活而上游无件=设计内静默；臂行陈=真断供。
锚定规则：**探测「链路活性」必须锚探针自己能区分的面**——拉取臂
动作产物（pull 表/心跳戳），不锚事件驱动的产出表。

## 处置

- fix/cal-feed-caliber（5e47e34d）：采集臂立 `md_calendar_pull` 拉取
  活性表+探针口径修——断更判据换锚 pull 行龄；facesnap 同步补册
  face:table:md_calendar_pull（11e23589）。
- 同类审查径：凡「feed 断更/鲜度」类探针先问产出是节拍型还是事件
  驱动型——事件驱动一律锚拉取侧，产出表只能当补充佐证。
- 升件处置遇「产出静默两可」件：若锚仍钉产出侧=口径病灶→要码修
  换锚非销件；锚已换而 pull 行陈=真断供照档处置。

## 证据

- check: 升件#10654 note（阈 24h 龄 24.1h「无高冲击事件窗」两可标）
  →修复后同窗同型件未再发
- result: 5e47e34d 部署（a3dafc3d 系批次）后 CALENDAR-FEED 探针口径
  =pull 锚，界缘假阳零复发
- ref: T-0032（无高冲击事件窗静默钦定）、升件#10752（pull 表入册）

## 反证条件

- check: pull 臂真断（md_calendar_pull 行龄>阈）而产出侧同步静默—
  —此时按「无事件窗」定性销=掩真断供
- falsifies_when: pull 表行陈超时仍被产出侧静默定性销件覆盖，证明
  处置径绕过了拉取锚——锚换面失效或消费面未用新锚。

## 适用边界

适用事件驱动型 feed（日历/Earn 产品/DCD 新品/venue 公告类）；
连续节拍型面（md_ticks/books5/depth5）产出静默本身即断供事实，
产出锚仍正确不适用本条。判「两可」件时锚侧先行核查——探针锚
钉错侧是口径病灶（要码修），不是数据面断流。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：探针口径判例非运行阈值指标——验收锚=
  升件零复发+pull 表活性行龄，非量表可测面）
