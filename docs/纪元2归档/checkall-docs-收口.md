# check_all docs-only 豁免——收口报告

> 施工：devin 施工席 @/opt/o-wt/checkall-docs｜分支 `chore/checkall-docs-only`｜2026-10-30
> 票面：`/opt/meta/docs/施工提示词-外部-checkall-docs豁免.md`

## 1. diff 摘

```
 docs/工程纪律.md    |  1 +
 scripts/check_all.sh | 50 +++++++++++++++++++++++++++++++++++++-------------
 2 files changed, 42 insertions(+), 9 deletions(-)
```

- `scripts/check_all.sh`：
  - 戳清理行加 `$STAMP.mode`——重跑先毁陈 mode 痕（防同 SHA 先跳后全留下假豁免痕）。
  - `go test 全谱` 段前插 diff 面判定块：merge-base 三阶退路
    （`origin/master`→`master`→`HEAD~1`）→已提交面 `git diff --name-only MB...HEAD`
    ∪工作树面 `git status --porcelain`（含 untracked，逐行剥 3 字符状态位，
    rename `a -> b` 双边同判）→逐件 case 判 `docs/*|skills/*|*.md`。
  - 全中白名单→stderr echo 判定依据（两层件数+merge-base SHA）+落
    `.git/check-green-<SHA>.mode=docs-only`；任一件非白名单→原段全谱照跑。
  - `CHECK_ALL_FORCE_FULL=1` 短路判定无条件全谱；merge-base 三阶全败或两条
    git 命令任一败→`DOCS_ONLY=0` fail-closed 走全谱。
- `docs/工程纪律.md` §4.3 补豁免说明行（见 §3）。

未动：戳语义（仍 `date -Is > check-green-<SHA>`）、pre-push 钩、其余门禁段、
CI/服务端面。

## 2. 三径实证

### ① 本支全谱径（.sh 非白名单→正常径）

HEAD=`22d4f04`（diff 面含 `scripts/check_all.sh` 非白名单件）：
输出 `── go test 全谱`（非跳过行）→ `go test ./...` 全绿
（`o/cmd/core 330.9s`、`internal/earn 291.5s`、`internal/projector 297.7s`、
`internal/research/*` 族、`internal/store/pgstore` 等全 ok）→
`✅ [check-all] 全链绿——戳落 /opt/o/.git/worktrees/checkall-docs/check-green-22d4f04f...`

### ② docs-only 跳过径（构造法）

本支 diff 恒含 .sh，跳过径实证须「HEAD-diff 纯 docs + 跑新版脚本」——构造：
探针树 `/opt/o-wt/checkall-probe`（支 `tmp/probe-docs-only`=origin/master
+`docs/tmp-checkall-probe.md` 一件 commit `99df2f35`），
`cd 探针树 && bash /opt/o-wt/checkall-docs/scripts/check_all.sh`
（脚本内 `cd toplevel` 落探针树做判定，判定逻辑取自施工版文件）：

```
── go test 全谱：docs-only 豁免跳过——已提交面 1 件+工作树面 0 件全中白名单
（docs/**、*.md、skills/**），merge-base=e70bfcd912c61c48e656a2baafe8f1d820e2110c；
CHECK_ALL_FORCE_FULL=1 压过
✅ [check-all] 全链绿——戳落 .../check-green-99df2f35...
```

`.mode` 实测：`cat check-green-99df2f35....mode` → `docs-only`。
其余门禁（build/vet/import/gofmt/env/rejects/migrations/bridge-audit）照常全跑全绿——
豁免只跳 go test 一段，码面结构律仍守。

### ③ FORCE_FULL 压过径

同探针树 `CHECK_ALL_FORCE_FULL=1 bash /opt/o-wt/checkall-docs/scripts/check_all.sh`：
判定短路→走 `── go test 全谱` 全量绿（同 99df2f35 戳重盖）。
且先毁陈 `.mode`——径②所留 mode 痕被 `rm -f "$STAMP.mode"` 清，
`ls check-green-99df2f35*` 余戳无 `.mode`：同 SHA 不残留假豁免痕。

## 3. §4.3 文面 diff

```diff
+- **本地 docs-only 豁免（业主令 2026-10-31）**：diff 面（已提交 merge-base∪
+  工作树未提交件）全中白名单（`docs/**`、`*.md` 任意层、`skills/**`）时
+  `check_all.sh` 免 `go test` 段、戳旁落 `.mode=docs-only` 留痕；
+  `CHECK_ALL_FORCE_FULL=1` 压过走全谱，merge-base/判定命令败 fail-closed 不豁。
+  CI 服务端 required-check 仍全谱兜真伪——豁免只省本地租金。
```

## 4. 绿戳 SHA

- 实证①：`check-green-22d4f04f49be72fcea57fc54dc7ae0f2f367060d`（本支全谱径）
- 实证②：`check-green-99df2f350d15b7cd2ecdd734112167aa49c74cfd` + `.mode=docs-only`
- 实证③：同 99df2f35 戳（FORCE 重盖，无 `.mode`）
- 推送头：`check-green-<本支终 commit SHA>`（本报告 commit 后全谱重跑盖戳——
  pre-push 验戳放行凭据）

## 5. 残留风险自列

- porcelain 未跟踪目录塌缩为 `?? dir/` 单行：目录以裸名判白名单——`docs/`、
  `*.md/` 形目录内含码件会算 docs-only。票钦 `docs/**` 语义本含 docs/ 下全件；
  残留仅 `.md` 尾目录名病态形，判低风险。
- porcelain 引用态路径（含空格/特殊符）带 `"` 前缀→白名单全拒=fail-closed
  走全谱，方向安全。
- worktree/本地缺 `origin/master` 视图时 merge-base 退 `master`/`HEAD~1`——
  落后基线只把面算大（多判进非白名单件→全谱），仍保守。
- 空 diff 面（HEAD=merge-base、树净）按空集全真判 docs-only→跳过——内容即
  master 已 CI 覆盖态，语义无害。
- 豁免只省本地租金：CI required-check 与服务端 master 保护不变，伪戳仍违例。
