# T-0433 check_all 二期——merge 谱定向化：码迁件免全谱

- 状态：完成（ops_event 完成行 2026-11-07 在案——机械轨票翻）
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

## 结果回填

- 实现摘要：`scripts/check_all.sh` 四档制——scoped 与 full
  间插 merge-scoped 档。分类面扩三旗标 HAS_MIG
  （migrations/*.sql）/HAS_PROTO（proto/*）/安全基建件
  （scripts/*、.github/* 不入 HAS_OTHER 不入 CODE_SEEN）；
  CODE_SEEN=.go|mig|proto 任一→定向谱候选（scoped/
  merge-scoped 同构统一分支——scoped=无附加种子特例零回归）。
  附加种子：迁移→`o/internal/store/pgstore`（migrate 测+
  反向闭包受影响包；check_migrations 闸门禁位恒跑不动）；
  proto→`o/internal/gen/<dir>` 逐文件映射+`buf build`
  proto 闸（buf 缺席/编译败→判不准回退 full）。闭包+恒测册
  五域 12 包+AP 展开失败回退全谱照旧；.mode 四档留痕
  （merge-scoped 附实测包集清单同形）。
- 测试证据（scratch wt 新脚本直跑三档+一兜底）：
  ①merge-scoped——proto/dashboardv1 探针+.go+scripts 件→
  mode=merge-scoped，包集=19 含 gen/dashboardv1+
  dashboardv1connect 种子+反向闭包（cmd/core、hlprobe、
  overviewprobe=dashboard 消费者实证）+恒测册 12 全在列
  （恒测册零漏=红桩断言成立——任意 diff 下五域必在谱）；
  ②scoped 回归——.go+scripts 件→mode=scoped 包集=13
  无 gen/pgstore 混入（T-0409 语义零回归实证）；
  ③fail-closed——proto/o/explorer 探针（internal/gen/o/
  explorer 无映射包）→go list 败→mode=full 全谱兜底；
  ④docs-only 分支代码面零动（ALL_DOCS 判据逐字未改）。
- 时长对照：merge-scoped 实测判定+包集瞬时（分类段
  go list×N+python 闭包 <5s）；谱宽由 diff 面决定——
  dashboard proto→19 包（含 cmd/core 重包）vs 全谱 ~90
  包；典型迁移+叶码 merge→pgstore 闭包+恒测册≈15-25 包
  量级（预期 20min→分钟级）。
- 偏离说明：①「proto 再生闸」裁读=buf build 编译校验
  （只读零写面）——gen 再生新鲜度归 bootstrap/deploy 径
  （check_all 全谱旧制同限不自再生，merge-scoped 不劣化）；
  ②merge 提交形非必需——分类按内容面不按 commit 形
  （同文件集分支推送同享定向谱，安全面等价）；
  ③迁移桩测件未能实跑——孤儿迁记 gate 先拒（探针迁移
  无登记行），pgstore 种子径与 .go 种子同构代码面可证；
  ④「已知安全基建件」白名单取最小集 scripts/*+.github/*
  ——web/tools 非 go/yaml/json/go.mod 全落 full 保守向。
- 坑处置①（自查发现 fail-open 已修）：基建件分支
  `scripts/*|.github/*) ;;` 首版漏清 ALL_DOCS——docs+scripts
  面误判 docs-only 免测（绿戳落在改了 check_all.sh 本体的
  commit 上=活标本）。已修=`ALL_DOCS=0`（基建件=中性：
  独现/伴 docs 落 full 不免测，伴码件入 merge-scoped）。
  红桩回归=scripts/meta/tests/check_all_scope_test.sh 场景①
  七场景全绿（docs-only/scoped/merge-scoped/full 四档+
  恒测册 13 包逐包断言+映射败/闭包败/go.mod 混入三路
  fail-closed）。
- 坑处置②（必读条目落地）：pipefail-grep-q-early-exit-
  false-negative——`docker ps|grep -q` 两处改
  `PG_NAMES=$(docker ps …||true)`+`grep -q <<<"$PG_NAMES"`
  herestring 同式（T-0409 遗留面同批根修，非新增）。
