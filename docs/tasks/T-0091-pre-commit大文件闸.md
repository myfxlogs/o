# T-0091 pre-commit 大文件/二进制闸

| 字段 | 值 |
|---|---|
| 状态 | 完成 |
| 领取 | devin-swe2-1025 @ 2026-10-25 |

> 病灶两犯在案：fillsprobe 16MB（T-0085）、core 28MB（T-0086）——`go build` cwd 产物
> 被 `git add -A` 扫入提交。工程经验库「go-build-cwd-artifact-swept」条目在档。

## 钦定规则

`scripts/hooks/pre-commit` 增闸：staged 文件满足任一即拒——
① 体积 >1MB 且后缀不在源码白名单（.go/.ts/.tsx/.js/.json/.md/.sql/.proto/.html/.css/.yaml/.yml/.toml/.svg/.png/.txt）
② `file` 判定为 ELF/Mach-O/PE 可执行体（白名单后缀豁免）
拒绝输出须点明文件+体积+修法（-o /tmp 或 .gitignore）。

## 验收锚

- 构造 staged 大文件 → commit 被拒且提示可读
- 正常源码提交零误伤

## 结果回填

**实现摘要**（devin-swe2-1025 @ 2026-10-25）：
- `scripts/hooks/pre-commit` 新增「11b. 大文件/二进制闸」：遍历 `git diff --cached --name-only --diff-filter=ACMR`，白名单后缀（.go/.ts/.tsx/.js/.json/.md/.sql/.proto/.html/.css/.yaml/.yml/.toml/.svg/.png/.txt）直放；非白名单走 `file -b` 判定——ELF*/Mach-O/PE32(+) 可执行体即拒；其余体积 >1MB 即拒。
- 拒绝输出逐件点名 文件+体积/类型+修法（`-o /tmp` 或 .gitignore），exit 1。

**测试证据**（/tmp 沙盒仓全钩实跑）：
- staged 12.7MB Go ELF binary（无后缀）→ 拒，输出 `probe （可执行体: ELF 64-bit LSB executable...）`
- staged 2MB `big.dat`（非白名单后缀非可执行体）→ 拒，输出 `big.dat 1953KB（非源码后缀大文件: data）`
- staged 2MB `big.md`（白名单后缀）→ 放行（两判全豁免）
- staged `a.go` 正常源码 → 全闸 `OK: pre-commit gate passed` 零误伤

**偏离说明**：闸插位 11b（11 与 12 之间），不重编号存量门禁——新增零 churn。
