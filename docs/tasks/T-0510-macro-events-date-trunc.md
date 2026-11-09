# T-0510 根修：macro_events期首日恒0——event_date<DATE×timestamptz界截断(周一weekly/月1monthly系统性失真)

- 状态：候选（T-0393 起草拍自动起草——非已批，COS/DM 过目才进派发队）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `根修:report-macro-events-date-trunc`，草案锚 fin_ticket_draft#44）

## 病灶（升件行证据引）

病灶签名：`根修:report-macro-events-date-trunc`（kind=探针类 `根修`，key=病灶键 `report:macro_events:date-trunc-boundary`）

末行升件原文：
> 外审席实证:date型列vs timestamptz界,PG截界至date→周一BoundsOf weekly Start=今00:00 End=now时窗内行全排count=0;桩TestZZProbe双行在库查询0实证。生产面:每周一weekly/每月1日monthly macro_events恒0。阻挡面:check_all全链红→一切master推滞留(T-0469裁过件/T-0509 merge排队)

证据引（ops_event 行 id）：

复核查询式：
```sql
SELECT count(*) FROM md_macro_event WHERE event_date>=$1::date AND event_date<$2::date
```

## 疑似根域

internal/ledger/report/report.go ~L101 macro_events计数行——event_date(DATE)<timestamptz界被PG截为date→期首日=End日行全排。修法:界统一::date化(End+1天保半开)或<=End::date;锚=TestReportMacroEventsCount周一红(非周一绿)

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 明确文件路径或目录清单（批准前必填——T-0413 派单依赖闸据此对拍）；
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

（施工写：实现摘要 / 测试证据 / 偏离说明）
