# T-0091 pre-commit 大文件/二进制闸

| 字段 | 值 |
|---|---|
| 状态 | 完成（判过并码在案，11-01清账翻面） |
| 领取 | devin-swe2-1025 @ 2026-10-25 |

审计席：headless @ 2026-10-31

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

## 审计注记（headless 逐锚对账 @ 2026-10-31，收口 088b1b04）

- 锚① staged 大文件被拒+提示可读：审计独立沙盒（/tmp/t91audit，scripts/ 整树拷入 + core.hooksPath 全钩实跑）复证——staged `/bin/ls` ELF（无后缀 probe）→ RC=1，输出 `probe （可执行体: ELF 64-bit LSB pie executable...）` 且头行点明 `-o /tmp`/`.gitignore` 修法；staged 2MiB `big.dat` → RC=1，输出 `big.dat 2048KB（非源码后缀大文件: data）`。两形态均拒、逐件点名 ✓。
- 锚② 正常源码零误伤：同沙盒 staged gofmt-clean `a.go`（3 行）→ 全闸 `OK: pre-commit gate passed` RC=0 落 commit ✓；2MiB `big.md` 白名单后缀 → 放行（钦定豁免语义）✓。
- 规则面核对：白名单 ERE 与票面 15 后缀逐字一致；`--diff-filter=ACMR` 覆盖新增/拷贝/改/改名；`core.quotepath=off` 中文路径不破匹配（门 7 同例）；可执行体判定 `file -b` 前缀 ELF*/含 Mach-O/PE32(+) 与 `file` 输出族对齐；`file` 缺席降级 ftype="?" 仅退回体积判不误杀；11b 纯插入位实证零 churn（diff 仅 +27 行未触存量）。hook mode 100755、闸已入 origin/master（088b1b04 为祖先）✓。
- 注记①（字面差）：可执行体分支拒绝行只带 文件+类型，未带体积——钦定「文件+体积+修法」字面上的 size 字段缺；类型串已点名身份且修法齐，判可读达标，建议后续补 `$size` 对齐字面。
- 注记②（残余面，已实证）：闸以 `[ -f "$f" ]`+`stat`/`file` 判 **worktree 字节**而非 staged blob——沙盒实证 staged ELF + worktree 改写小文本 → commit 落库（ELF blob 46c903a 落 index）；`git add` 后 worktree 删/改文件亦可绕（`[ -f ]` skip）。主病灶径（`git add -A` 扫入时 index==worktree）全覆盖，错位绕径需刻意构造或罕见时序；若要对齐「staged 文件」字面，修法=`git show ":$f"` 取 blob 喂 `file -`/`wc -c` 判。报管理席酌处，非本票锚违例。
- 回填诚实性对拍：回填四条测试证据与审计独立复跑输出形态相符（1953KB vs 本席 2048KB 系 dd 计数差，语义同）；实现摘要逐句与收口 diff 一致，无夸大。
