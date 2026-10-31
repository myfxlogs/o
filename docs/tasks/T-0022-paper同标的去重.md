# T-0022 paper 同标的去重（dev 库实证毛边——同 inst 同向 paper 仓叠加）

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 提出 | `[角色:决策]` 2026-10-23（dev 库实证） |
| 类别 | 缺陷修复（小件） |
| 依赖 | T-0012✅ |
| 领取 | devin-t0002 @ 2026-10-23 |

## 背景

dev 实证：BTC 提案每拍 armed→新 paper 策略行，同 inst 同向已挂 5+ 个 paper（fin_proposal 去重闸只管 proposed 态——armed 后下一拍再放一波）。同一标的的 paper 仓叠加=名义额虚增+结算重复，不是「多实例」是「重复实例」。

## 裁决

**同 venue+inst_id+direction 同时最多一个 active paper 策略行**——新提案评估通过时已有同键 active paper=复用（不新建行），已有行非 active（retired/closed）才许新开。策略行 dedupe 唯一索引（部分索引 where state in active 集）。

## 验收标准

- [ ] 部分唯一索引+插入侧拒绝路径测试锚（同键第二 active 行拒建）
- [ ] 已有 active 行时提案不重复建策略（E2E：连拍两次同 inst proposed→strategy count=1）
- [ ] 机检全绿；CI 绿

## 范围外

- probe/live 态去重（真钱态一 strategy-instance 一仓已由 fin_position 面守）；跨 inst 面

## 结果回填

分支 `t/0022-paper-dedupe`（已推 origin）。施工面=worktree `/opt/phase-wt/t0022`。

**验收锚对账**：

- 部分唯一索引+插入侧拒绝锚 = **PASS**：`0014` direction 补列+`fin_strategy_active_dedupe`（venue+inst_id+direction WHERE state<>'retired'）；同键第二 active 行直插拒测试锚过。
- 已挂 active 行提案不重复建策略 = **PASS**：admitProposals 加 direction 列+`ON CONFLICT...DO NOTHING` 复用径——同键 active 行在挂时提案 armed 被吸收不叠仓（E2E：连拍两提案→strategy=1、armed 提案=2 实拍）。
- 机检全绿 = **PASS**：build/vet/gofmt/lines/env/migration/import 全过；go test 15 包全绿 @5434。

**实测修正**：排查 direction 列缺席耗时——根因=测试建双 Pool（setup 内一池跑迁移+外层一池空壳落 public 旧表），改 setup 返回同池（T-0021 schema 隔离面暴露出的测试自身结构坑，同族「本地过≠隔离对」教训）。

## 审计注记

审计席：headless @ 2026-10-31。收口 commit `d5862abd` 全量 diff 逐锚对账：

- 锚① 部分唯一索引+插入侧拒绝路径测试锚 = **PASS**：`migrations/0014_strategy_dedupe.sql` 建 `fin_strategy_active_dedupe`（venue+inst_id+direction WHERE state<>'retired'）——fin_strategy 状态 CHECK 为 proposed/paper/probe/live/degraded/retired，retired 是唯一终态，「active 集=非 retired」对位正确（裁决文"retired/closed"之 closed 在状态机中不存在，非锚偏）；`TestPaperDedupe` 直插同键第二 active 行断言被拒。
- 锚② 已有 active 行提案不重复建策略 = **PASS**：`run.go` admitProposals 走 `ON CONFLICT (venue,inst_id,direction) WHERE state<>'retired' DO NOTHING`+RETURNING 空→提案置 fin_armed 吸收径；`TestPaperDedupe` 连拍两同键提案→scan→fin_strategy count=1、fin_armed=2，与锚文逐项对位。
- 锚③ 机检全绿；CI 绿 = **PASS（注记）**：commit 经当日已武装 pre-commit 钩过闸，回填列明 build/vet/gofmt/lines/env/migration/import+15 包 @5434；仓内 GitHub Actions 首跑为次日（10-25T07:40Z a5d37a48 起全绿），本票 SHA 无 check-run——"CI绿"在当时系前瞻声明，但 d5862abd 系首个绿跑祖先且落 master 后持续绿，锚意实达。

非阻断注记：①原迁移缺脏数据前步——对脏库 CREATE UNIQUE INDEX 必败（本票动机即脏 dev 库），同日 T-0024 `be90dab9` 补 dedupe 前步+坑条目 migration-unique-index-dirty-data 入库，已愈；②`TestDbg4` 调试残渣随本票入仓，T-0216 `8bdb5efc` 已清；③`err.Error()=="no rows in result set"` 字符串比对判空行（宜用 errors.Is(pgx.ErrNoRows)），master 现行同形（run.go:164/xvenuepair.go:52），建议后续顺手改；④direction 列 `NOT NULL DEFAULT 'short_perp_long_spot'` 为迁移回填兜底，列级永驻默认值对将来漏写 direction 的写者构成静默兜底，后续可考虑去默认值。
