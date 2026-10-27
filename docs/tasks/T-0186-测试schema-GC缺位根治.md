# T-0186 测试 schema GC 缺位根治

> 立单：决策席 2026-10-27（T-0174 施工收口上报 → 决策裁定立项）
> 域：dev `o-postgres-test`（5434）测试实例
> 量级：中件

## 病灶实证（T-0174 施工席取证回填）

- `internal/store/pgstore/testutil`（或同族 helper）每测试建 `t_*`
  schema，**用后不清**——实例积 18,365 schema / ~32GB 才发现。
- 前身 GC 脚本按 `arb_*` 库名+arb 角色写成，与 O 拓扑脱钩，
  cron 连日 `auth failed`（脚本活着但从未成功过=半死人形态）。
- 本次应急已逐条 DROP 清至约 7 千 schema，**但根源未修**：
  测试继续跑就会再积。

## 钦定修法（二选一，施工判）

**甲案（首选）**：testutil 自清——`t_*` schema 在 `t.Cleanup`
或连接归还钩里 DROP CASCADE。测试内的 schema 生命周期归测试
自己管，零外部依赖。

**乙案**：O 拓扑 GC 重写——`scripts/gc_test_schemas.sh`：
只认 `t_*` 前缀 + 创建时间 >Nh + 库=o_test 三重钉（防误杀），
挂 dev cron 日拍。前身脚本入档不留 cron。

**禁线**：DROP 范围只许 `t_*` 前缀 schema——`public`/`fin_*`/
`md_*` 白名单永不进 DROP 集。

## 验收锚

1. 测试跑后 `t_*` schema 不残留（甲案）或日拍清零（乙案）；
2. `o-postgres-test` 体积增速回到正常水位（T-0181 盘漂哨兵
   现每日机械报，作长期观察面）；
3. 前身 arb_* GC cron 条目移除或钉死注释。

## 归档/参考

- 经验库条目：`docs/工程经验库/条目/test-schema-per-test-no-gc-instance-suffocation.md`
  （T-0174 施工已入库）

## 状态

- [x] 立单
- [ ] 施工
- [ ] 验收投产
