# T-0436 DM 批 merge 一戳——洪峰期连合盖顶协议

- 状态：施工中（业主令 2026-11-07——「立」；前置账=洪峰日
  闸耗分解实证：5件merge×旧制全谱~20min≈1.7h 纯串行，
  批戳可省 80min/洪峰）
- 领取：claude-t0436-ext @ 2026-11-07（外派施工，O-Dev 径
  ——决策席 SKILL 班次节改协议，docs 面不占 DM 钱径额度；
  支 t/0436-dm-batchstamp@origin/master，票文自本地 master
  立票提交 2f8f13f0 cherry-pick 带支——origin/master 尚未
  含票文，master 上的票面状态滞至合并时由决策席统一翻）
- 优先级：P1——裁决洪峰结构性剩余：merge→戳→推 串行×N
  是 0433 没切到的另一半账
- 收口移交归：外审径（外派外审外部署成例）

## 病灶

DM 裁决班现行序=逐件「审→merge→check_all→推→下一件」：
N 件待验=N 个串行 merge 戳。但 pre-push 闸只认「HEAD 绿
戳」——中间 merge 的戳在下一 merge 落地那一刻即作废
（戳绑 SHA 坑条目实证），**N-1 个戳是纯废功**。洪峰日
（外派 5 件同拍收口 2026-11-07 实证）串行戳=纯串行税。

## 面（钦定）

决策席 SKILL 班次节加「批merge盖顶」协议：

① 洪峰判据：待裁件≥2 且同属「merge 后要戳」面（非
   docs-only 直推件）→进批径；单件仍走件件戳旧径零回归；
② 批序：逐件审+裁（判照一件件出，裁记照落票面）→
   过件连合 master（merge×N 中间不跑 check_all）→
   末件合完跑一次 check_all 盖顶→推→逐件翻归宗；
③ **批戳红处置**：顶戳红→bisect 二分定位撞车件
   （各 merge 件自带工人绿戳在手，撞车定位只判
   「哪两件相碰」非「码对不对」）→肇事件退件回施工，
   其余重新盖顶；bisect 步序写死禁临机发挥；
④ **钱径件禁批**：钱径/风控/迁移危险面 merge 永远件件戳
   ——批径只许工程/数据/文档类进（闸件含 scripts/** 面
   可进批但标注意，它决定全谱时长）；
⑤ 裁记形制：批件票面裁记标「批径N件盖顶@sha」——
   读戳面对得上（T-0397 同形）。

## 验收

- 协议文本落 skills/决策席/SKILL.md 班次节+桩例（批径
  判据/盖顶序/bisect 步序文字可机读——伪件流程重演一遍
  验证步序无歧义）；
- 禁线断言文字面：钱径件混入批径=协议违例项写死；
- 绿戳对口+全链（本票自身=skills/docs 面→docs-only 档
  实证快档）；坑处置随 commit。

## 备注

与 T-0433（单件戳时长）×T-0435（锁/库争用）三件套咬合：
本票砍的是「件数×戳」的件数因子。批径跑顺后归宗洪峰
理论下界=1 戳/批而非 N 戳/批。

## 结果回填（claude-t0436-ext @ 2026-11-07）

**实现摘要**：`skills/决策席/SKILL.md` 班次节插「批merge盖顶」节
（b50f2ca4，不另起件，83 行）：治面锚戳绑SHA坑条目；①洪峰判据三断言
（待裁≥2／剔禁批面+docs件后≥2 且面判径=票支 diff 机对④清单／基线断言
HEAD==origin/master）；②批序六步写死（整批持 dm_repolock 一锁到裁记
落纸、逐件审裁零批化、merge×N 中间零 check_all、一次盖顶、推前
HEAD==戳SHA 断言败=升件不推不弃、裁记落纸必在推毕走 docs-only 快档
补戳、翻归宗归壳轨）；③批戳红 bisect 五步写死（清册=
`git log --reverse --merges` 对票／对半切 k=lo+⌈(n−lo)/2⌉ 判色=
CHECK_ALL_FORCE_FULL exit code／gen 错位红=升件不裁／处置=reset 基线
re-merge 非肇事件+退件驳回行返修提示词／全程持锁禁推禁部署）；④钱径件
禁批面清单写死+清单外按禁批计单向兜底；⑤复核对账（@sha 对
check-green-<sha>+.mode，T-0397 同形；批顶戳不替代件戳）；⑥幂等复入；
桩例（3 件批伪件全步序重演）。

**伪件重演证据（验收①后半）**：bisect 步序机演 PASS——协议③步序直译
模拟（scratch 件 /tmp/t0436_bisect_sim.py，不入库不碰闸脚本）：n=2..8
×全肇事件位 j 全组合收敛正确、k 永不出区间（lo<k<n）且永不重测已知
端点、判色跑数上界=⌈log₂n⌉（n=5 洪峰最坏 3 跑 vs 旧径 5 跑）；桩例
n=3,j=3 路径=k=2 测 S₂ 绿→lo=2→差1→M₃=T-C 与文一致。桌面歧义修三处：
节内交叉引用消歧（「步②」→「③步②/步②3–5」）；ops_event 追记行删
（避免自造新事件 kind 触账链闸面——票面⑤只要求票面裁记）；补 gen/buf
编译错位红=环境错位升件不裁 fail-safe 线+批剔空仍红升件终止界。

**验收锚对账**：
- 协议文本落班次节+桩例（批径判据/盖顶序/bisect 步序文字可机读——
  伪件流程重演验证步序无歧义）：**PASS**——锚=SKILL.md「### 批merge
  盖顶」节（①判据三断言/②批序六步/③bisect 五步全带命令级写法）+
  桩例节；机演 PASS（上段，判据/步序逐条机读）。
- 禁线断言文字面（钱径件混入批径=协议违例项写死）：**PASS**——锚=④
  节「钱径件混入批径=协议违例入案（伪戳同级）」+禁批面清单写死
  （migrations/risk/guard/ledger/trading/venue/mt5/凭证形件）+清单外
  按禁批计单向兜底。
- 绿戳对口+全链（本票自身=skills/docs 面→docs-only 档实证快档）：
  **PASS**——收口 check_all mode=docs-only 绿戳（尾行：`✅
  [check-all] 全链绿（mode=docs-only）`）；坑处置随 commit：无新坑
  条目——戳绑SHA坑=既有条目 `check-all-stamp-binds-head-sha` 且正是
  本协议治面，批序已按其正序律写死（盖顶先于推、裁记落纸走快档补拍
  =坑条目处置款钦定径）；施工全程未出新坑（领单推被 pre-push 拒一次
  =既有坑条目时序律正常工作，非新坑）。

**机检**：check_all 全链绿 mode=docs-only（exit 0，戳落
check-green-<收口SHA>）；本票未触 .go/迁移/proto 面，其余闸随 docs-only
档恒跑段全过（import/gofmt/env/rejects/config-write-domain/migrations/
bridge/tolerate/ops-chain）。

**涉及面**：`skills/决策席/SKILL.md`（班次节插节）、本票文、
`docs/prompts/T-0436-外部施工提示词.md`（立票 cherry-pick 带支）。

**范围外确认**：未越单——闸脚本（check_all/git-hooks）、壳脚本
（dm_*）、码面/迁移/proto 零触碰；协议文本明写「纯协议节零闸脚本改动」。

**偏离说明**：两处施工裁量=票面「面」钦定的自然细化非偏离：
a) 禁批面清单按 `docs/架构约束.md` §2 六域命名落路径（票面只点名
「钱径/风控/迁移危险面」）；b) 裁记落纸时序钉死在推毕（票面⑤只写
形制未写时序——戳绑SHA坑条目正序律推出，其处置款明文钦定 docs-only
收尾件趁戳在案后 commit 快径补戳）。

分支: t/0436-dm-batchstamp（已推送=已交付）
移交: 决策席（外审径——外派外审外部署成例）
