---
schema_version: 2
id: queryrow-placeholder-missing-arg-silent-skip
kind: pitfall
risk: high
status: verified
keywords: [pgx, QueryRow, 占位符, 缺参, 静默吞错, 幂等查, 提案径, err==nil合并条件, 死代码假活, T-0377]
scope: [go, pgx/v5, 数据写入径, 幂等分支, 评估→提案链]
summary: QueryRow SQL 声明 $1 却不传参——pgx 报 expected N got 0，若嵌在「err==nil && 条件」合并分支则整段写径静默跳过零日志零拒因，下游表恒空且无人知。
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: [KB维护]
evidence_refs:
  - type: commit
    ref: T-0377（本条目同 commit）
  - type: doc
    ref: internal/research/hypothesis/xcarry.go（提案 count 查询）
  - type: server_fact
    source: postgres
    query: SELECT count(*) FROM fin_proposal p JOIN fin_opp_eval e ON e.id=p.eval_id WHERE e.template='xvenue-carry'
    observed_at: 2026-11-06T01:10:00Z
---

## 事实/症状

T-0377 链路实查：xvenue-carry 评估行 10-26~10-30 期间正常落（verdict 一度
proposed），但 `fin_proposal` 零行——评估/提案之间断链。代码面上
`xvenueCarryEval` 的提案幂等 count 查询写着 `WHERE eval_id=$1`，调用处
**没传 evalID 参数**：pgx 报 `expected 1 arguments, got 0`，而该查询嵌在

```go
if err := r.pool.QueryRow(ctx, `…eval_id=$1`).Scan(&n); err == nil && n == 0 {
    INSERT…
}
```

的合并条件里——err≠nil → 整个 INSERT 分支静默跳过：零日志、零 reject
计数、零 panic，下游表恒空。

## 根因或策略解释

1. **占位符与实参解耦无编译期检查**——SQL 字符串里的 `$1` 与 Go 实参表
   各自独立，少传一个实参编译器不红。
2. **`err == nil && n == 0` 双条件合并把「查询败」与「无行」压缩成同一
   出口**——查询败=不写（fail-silent 而非 fail-closed），分支退出无任何
   可观测痕迹（rejects.Inc 也没挂）。
3. **上游掩护放大**：该 bug 在 execfam 前置拒（exec_family_missing）下游，
   前置拒使 verdict 恒 rejected → 提案分支从未被执行到 → bug 不可见。
   「上游闸挡住 ≠ 下游代码对」——闸一开（T-0377 登记 execfam）断点立即
   变成新首断。

## 处置

- 根修：补传 evalID（T-0377 同 commit）；
- 查询 err 面显式留痕：err≠nil → rejects.Inc + log，不再靠合并条件吞；
- 测试钉：TestExecFamilyRegisteredChainProceeds 断言「建族后提案立 1 行」
  ——此前链路无此断言，静默跳过无测试红。

## 证据

- check: 测试桩直捕 err（临时 DEBUG 输出）→ `expected 1 arguments, got 0`
- result: 修复后同 fixture 提案立行（fin_proposal count=1），hypothesis 全包绿
- ref: T-0377 施工分支 commit（本条目同 commit）

## 反证条件

- check: `go vet ./...` 不红（vet 不查占位符数）——反证须运行期
- falsifies_when: 修后 xvenue-carry 提案仍恒零且 err 面无留痕（根修未生效）

## 适用边界

- 全仓 `QueryRow(ctx, sql)` 带 `$N` 占位符的调用面通用——同类形态可在
  「幂等 count → INSERT」与「SELECT → 条件写」分支扫查（本仓第二处：
  无——T-0377 实查时 grep 复核）。
- 教训入纪律面：**err 合并条件禁吞查询败**——写径分支的 err 应显式留痕
  （rejects/log）或上抛，不得与业务条件同出口静默。
