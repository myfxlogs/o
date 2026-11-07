---
schema_version: 2
id: neutral-case-branch-fail-open-whitelist
kind: pitfall
risk: high
status: verified
keywords: [check_all, 测谱分级, docs-only, fail-open, case分支, 白名单, ALL_DOCS, 豁免闸]
scope: [scripts/check_all.sh, bash 分类器/白名单判据]
summary: 分类器 case 里给「中性件」开 `pattern) ;;` 空分支=fail-open 地雷——中性件不置任何旗标，若忘同步清掉正向豁免旗标（ALL_DOCS 类），docs+中性件混合面被误判全豁免档=零测试假绿；中性件语义必须显式双向声明（不算A也不算B）
first_seen: 2026-11-08
---

## 病灶

`check_all.sh` 测谱分类器（T-0409 三档→T-0433 四档）按
`git diff` 路径集 case 分支置旗标：

```bash
docs/*|skills/*|*.md) ;;              # 豁免档：不动任何旗标
*.go) HAS_GO=1; CODE_SEEN=1; ALL_DOCS=0 ;;
scripts/*|.github/*) ;;               # ← T-0433 首版：中性件空分支
*)   HAS_OTHER=1; ALL_DOCS=0 ;;
```

「中性件」（安全基建件——不算码不算清单外、可伴码件入定向谱）
用空分支表达，**但忘了它同样不是 docs**——`ALL_DOCS` 保持 1。
结果：docs+scripts 混合面 → `ALL_DOCS=1` → docs-only →
**go test 整段免跑**，绿戳照落。实证：T-0433 自身 diff
（check_all.sh+票面 md）即中此——绿戳盖在改了闸本体的
commit 上，docs-only 档跳过全谱测试。

## 修法

中性件分支必须**显式双向**：

```bash
scripts/*|.github/*) ALL_DOCS=0 ;;   # 不算docs→不免测；HAS_OTHER不置→可入定向谱
```

判据：任何「既不置正向旗标也不置负向旗标」的 case 分支，
都要问——它是否会让某个全有/全无判据（ALL_DOCS、ALL_X）
在混入时仍成立？会→补清旗标。

## 实证

2026-11-08 T-0433：check_all 首轮跑出 `mode=docs-only` 戳，
.diff 面含 `scripts/check_all.sh`——反向核对分类器发现
`scripts/*` 空分支漏清 `ALL_DOCS`。修复后
`scripts/meta/tests/check_all_scope_test.sh` 场景①
（docs+scripts→full）为红桩回归锚，七场景桩测全绿。

## 回改点

- 写分类器/白名单 case 分支时，中性件=显式清正向旗标，
  禁空 `;;`；
- 「全 X 才 Y」判据（ALL_DOCS/ALL_TESTED 类）配套审：
  每个不进入 Y 的分支是否都清了全 X 旗标；
- 豁免档（docs-only 类）是 fail-open 高发面——凡新增
  「可豁免」路径分类，必配「豁免面+非豁免件」混合桩测。
