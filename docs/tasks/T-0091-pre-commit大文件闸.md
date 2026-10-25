# T-0091 pre-commit 大文件/二进制闸

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
