# T-0444 探针升件自动起草：观测盲@CALENDAR-FEED

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:CALENDAR-FEED`，草案锚 fin_ticket_draft#3）

## 病灶（升件行证据引）

病灶签名：`观测盲:CALENDAR-FEED`（kind=探针类 `观测盲`，key=病灶键 `CALENDAR-FEED`）

末行升件原文：
> 日历 feed 断更：末行 sent_at=2026-11-04 05:10Z 龄=58.1h（阈24h——断更形含「无高冲击事件窗」两可如实标，对拍归人肉/上游活性面；探针只报不修）

证据引（ops_event 行 id）：#2823

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='CALENDAR-FEED' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/research/ctx/（ffcal 日历采集面——sent_at 断更 58h 根域：真断供 vs 无事件窗两可甄别）
   scripts/meta/（ctx 采集拍/探针阈面）；
2. 病灶根因定位与根修（签名面病灶本体——不超签名外衍）；
3. 桩测钉死断言（病灶存在断言修前红+同断言修后绿——T-0388 修复类先红桩律）；
4. 生产谱面零误报审计。

## 验收锚（骨架——DM 裁时补足）

① 桩测断言全绿（红/绿双点证据落结果回填）；
② 探针同窗不再报同签名病灶（修复后探针行随停）；
③ check_all 绿戳对口；并入发窗批。

## 红线（继承模板——不改不删）

- 只修本票签名病灶——不超单施工、不碰范围外文件；
- 不部署、不并 master——收口回决策席常径裁；
- 探针只报不修——本票修病灶本体，禁顺手改探针阈/签名面/类册；
- 钱径/凭据域/真单径语义禁碰。

## 结果回填

- 状态：待验
- 实现摘要：
  1. **两可甄别结论**（票面钦定项——真断供 vs 无事件窗）：CALENDAR-FEED
     两升件（#2823 11-06 15:17Z、#3935 11-08 04:24Z）=探针钦死的「无高
     冲击事件窗」两可形，feed 本体健康——生产取证三通道交叉：
     ① ops_event 全量零 `读败` 升件（拉取/非200/解包/空返四相皆无——
     feed 拉取全程 200 无退避档记录）；
     ② SG `md_calendar_push` 实读：NFP 簇 11-06 17:32:53-55Z（事件 18:30Z
     窗内 T-58min）、BOJ Ueda 20:11Z、NZ 选举 11-07 01:59Z（窗 01:45-02:45
     内）——全部 High 事件均在正确预推窗落行；
     ③ FF 周历 wayback 逐日快照（11-02~11-06）：High 事件表逐日在册
     无缺——断更窗内无遗漏应推事件。
     11-04 05:09:57-05:10:00Z BOE 五行特殊形=T-0326 部署验收钦定干跑
     注入（`calendar.lead_minutes` 60→2200 热启→真推→复原 60——票面
     已录「自然 T-60 预推被幂等行抑制，信息前移非丢失」）。
  2. **签名面真病灶根修** `internal/research/ctx/collect.go`：ffcal→
     `md_ctx_fact` dedupe_key=url slug 无版本位——FF url slug 天生无
     日期（`11-us-unemployment-claims`/`86-us-ppi-mm` 类周/月循环事件
     同键恒存），次周同 slug 新发生撞 `ON CONFLICT DO NOTHING` 恒静默
     吞行=事实层断更（生产实证 OCI `md_ctx_fact` 行
     `11-us-unemployment-claims` payload 冻结在 Date=11-05-2026——下周
     四新发生永不落行）。同族 KB
     `dedupe-key-missing-version-bit-silent-stall`（T-0429 fred/
     frankfurter 同法已修，本件=ffcal 面收口）。修法=键 `slug:事件日期`
     （空 slug=键空=NULL 走 ins 旧径不互撞）；同日重拍幂等不变。
- 测试证据：
  - 红→绿双点：`TestFFCalDedupeNewDate` 修前红（dedupe_key 期望
    `claims:11-05-2026` 实=n0/次周新发生零行）修后绿；
    `TestFFCharset1252` 键期望随新键形同步（`pmi-revise`→
    `pmi-revise:11-06-2026`）。ctx 包 12 测全绿 19.1s（O_TEST_PG_DSN
    :5436 租约）。
  - 自测链：`go build ./...`+`go vet ./...`+gofmt+`check_migrations`
    七断言全绿。
  - 生产谱面零误报审计：dedupe 键加日期位只增不减——同日重拍仍冲突
    抑制；新行=新 id 喂 analyst `ctx_max_id` 水印增量（本意语义）；
    CTX-FFCAL 探针读 `max(received_ts)` 不受影响；eventbus
    md_ctx_fact 镜像按 id 水位照常；`md_macro_event` High 桥表走
    event_date 冲突键不涉本改。
- 偏离说明：
  ① CALENDAR-FEED 探针升件本体=钉死的「两可」形——事件荒窗（周末/
    无 High 事件段）行龄 >24h 必再报，探针阈/签名/类册红线内无可修
    面；探针语义是否收窄（如按上次成功拉取戳而非推送行龄判死）归
    决策席裁，本票不越权；
  ② scripts/meta/ 钦定面核查零改动——ctx 采集拍在 OCI o-research 服务
    内嵌 Run（无 meta 脚本件），探针阈面=probe.go staleRules 代码表
    非脚本，病灶不在该面；
  ③ 存量冻结行不回填——`11-us-unemployment-claims` 等旧行 dedupe_key
    无日期位、payload 停旧值，新键位起新行并行（append-only 语义，
    不 UPDATE 旧行——旧行=11-05 发生当日的合法历史快照）。
