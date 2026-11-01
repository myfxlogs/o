# T-0038 clOrdId 字符集根修——连字符非法（真单实勘 51000）

[角色:决策] 立 2026-10-24 | 优先级：**阻断级（真单实测炸出）**

| 状态 | 完成（判过并码在案，11-01清账翻面） |
| 领取 | zcode-glm @ 2026-10-24（worktree /opt/phase-wt/t0038，支 t/0038-clordid-charset） |

审计席：headless @ 2026-10-31

## 病灶（真单实证 exec_id=1）
- `flow.go` clOrdID=`ph-<exec>-<leg>-<nano>` 含连字符——**OKX clOrdId=大小写英数 only**（禁特殊符）
- 实跑回放：Place 拒（order 未建）→Resolve 同串亦拒 `51000 Parameter clOrdId error`→ambiguous fail-closed（钱零损——fail-closed 兜住实锤）
- NOX 实证口径：纯英数（"abc123" 样例+28 截断）

## 修法
- 格式改 `ph<exec>x<leg>x<nano>`（字母分隔位合法）或 `ph<exec><leg><nano>`——≤32 英数
- 测试锚：clOrdId 正则 `^[A-Za-z0-9]{1,32}$` 断言
- 经验库条目：venue 方言字符集账（clOrdId 禁特殊符——OKX 实拍 51000）

## 验收锚
- 生成串过 `^[A-Za-z0-9]{1,32}$`；flow 全测绿

## 结果回填（[角色:施工] zcode-glm 2026-10-24）

- 根修：flow.go 两处生成式改字母位分隔——`ph<exec>x<leg>x<nano>` / `ph<exec>rb<nano>`（英数 ≤32；连字符禁）。
- 测试锚：`TestClOrdIDCharset`（两形生成串过 `^[A-Za-z0-9]{1,32}$`）；fakeVenue 断言串随格式更新（"-rb-"→"rb"——TestRollbackFailedEscalates/LeggedRollback 复绿）。
- 经验库条目：`okx-clordid-charset-alnum-only`（schema v2——含通用律：venue 自由字符串面字符集先验+适配层生成式集中+首单前契约实拍）。
- 机检：build✓ vet✓ gofmt✓ test✓（全量 20 包零 FAIL）import✓（55/55）lines✓ env✓ 迁移面零改动（一行修无迁移）。

## 审计判决（审计席：headless @ 2026-10-31）

逐锚对账（收口 8a4f8f61 全量 diff 亲读+独立实跑复证）：

- **锚① 生成串过 `^[A-Za-z0-9]{1,32}$`** ✓：diff 内两处生产生成式
  `ph%dx%dx%d`（placeAndResolve）/`ph%drb%d`（rollback）字符面全落
  `[a-z0-9]`⊂锚正则——连字符除尽；`TestClOrdIDCharset` 锚实跑 PASS。
  长度面 execID int64 理论极值可破 32——现实序列量级远不及，且长度敞口
  为先前既有非本票引入，注记留痕不报阻。
- **锚② flow 全测绿** ✓：收口 commit 本席独立复跑
  `go test ./internal/trading/execution/ -count=1`——13/13 PASS 零 FAIL
  （PHASE_TEST_PG_DSN 实测库，含 rb 径 TestLeggedRollback/
  TestRollbackFailedEscalates——fakeVenue 判定串 "-rb-"→"rb" 随格式
  同步一致：主径字符集 [a-z0-9x] 不含 r/b 序列，"rb" 子串唯 rollback
  形可命中，谓词无歧义）。
- **修法三件套附证**：经验库条目 `okx-clordid-charset-alnum-only`
  （schema v2）在册，master 现行 flow.go 仍持修复形态（L106/L175）。
- **注记（非阻项）**：`TestClOrdIDCharset` 断言作用于测试内字面量
  Sprintf 复刻串而非生产生成路径——flow.go 格式串若回漂连字符该锚
  不复红（条目「生成面漂移即红」表述略溢）；生成式未抽公共函数为根
  因。锚文本义已达，质量面注记报管理席知悉。
