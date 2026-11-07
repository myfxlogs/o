# T-0433 check_all 二期——merge 谱定向化：码迁件免全谱

- 状态：已批（业主令 2026-11-07——「409二期外派单，太费时间了，
  都在等DM，越到后期项目越大越跑不过来」）
- 领取：外派施工（O-Dev 径——闸脚本面钦定全，不占 DM 钱径额度）
- 优先级：P1——吞吐瓶颈：DM 串行裁决×每 merge 全谱 ~20min，
  归宗洪峰实证（5件待验同拍砸桌，工期 2h+）
- 收口移交归：外审径（业主令 2026-11-07——「外派外审外部署，不去给DM做积压」：外派件闭环=外派施工→外审席裁→外部署，DM 只裁内联票；本票成例——后续外派票同此径）

## 病灶（2026-11-07 实证）

T-0409 三档已生效（docs-only/scoped 实证两推豁免），但**merge 场景
永远落 full**：码票必带 migrations/*.sql/proto/sh → HAS_OTHER 钉死
全谱。每张 merge 付全谱价，受影响包与无关包同跑——DM 裁决班的
闸耗时是「件数×全谱」线性涨，外派分流越成功洪峰越堵。

## 面（钦定）

① **新档 merge-scoped**（在 scoped 与 full 之间插档）：
   diff 面=.go+migrations/sql+proto+已知安全基建件的 merge 提交
   →跑「变更包+反向依赖闭包+恒测册+迁移闸+proto再生闸」，
   无关包 go test 免；
② 恒测册不可削：o/internal/{risk,ledger,trading,guard,
   portfolio/lifecycle} 等钱径/风控册**不论 diff 恒测**——
   定向省的是无关包不是安全面；
③ 迁移件定向：fin_/md_ 迁移→check_migrations+pgstore
   migrate 相关测+受影响包，非全谱；
④ **fail-closed 硬钉**：包映射/反向依赖/受影响面任何一步
   判不准→回退全谱，不 fail-open；scope 判据日志留痕
   （哪包进谱哪包免，戳旁 .mode 附实测包集清单——T-0397
   三档读戳面同形）；
⑤ 防窄化回环：scoped 判据误判漏测钱径包=红桩断言——
   票面钦定「构造一次漏测场景的桩测件」必过。

## 验收

- merge 谱实跑计时对比（同 diff 面 full vs merge-scoped
  耗时降幅实测值回填——预期 ~20min→分钟级）；
- 恒测册包零漏（变更面任意构造下 risk/ledger/trading
  册必在谱——红桩断言）；
- 判不准径构造（映射断/闭包炸）→落 full（断言）；
- docs-only/scoped/full 三档旧行为零回归（既有桩全绿）；
- 绿戳对口+全链；坑处置随 commit。

## 备注

二期只做闸耗时——DM 串行单点本体是另一结构性议题
（决策席并行化/裁决分席），另案挂不混票。
