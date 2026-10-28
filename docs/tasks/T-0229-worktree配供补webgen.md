# T-0229 worktree 配供补 web/src/gen——四次显形坑根修

| 项 | 值 |
|---|---|
| 状态 | ✅ 完成 |
| 优先级 | P1 |
| 性质 | 基建坑修（scripts 面） |
| 立项 | `[角色:决策]` @ 2026-10-28——**业主直令「先把坑填了吧」**：决策席跨角施工（基建脚本非产品码），验收=机检+实证回填 |
| 涉及面 | `scripts/provision_worktree.sh` |

## 病灶

`provision_worktree.sh` 只备 `internal/gen`+`web/dist` 两件，漏 `.gitignore` 第三件 `web/src/gen`——
新 worktree 里 vite dev/build 缺 `../gen/dashboardv1/core_pb` 引用→SPA 空白/构建炸，
四次显形在案（KB 条目 worktree-provision-missing-web-gen/worktree-fe-rebuild-gen-gap 同源）；
dist 快拷贝快照无刷新语义——worktree 内任何 `go build` 嵌入速冻前端，混装风险面在。

## 钦定修改

1. need 检查+拷贝+慢径+终检四处同步加 `web/src/gen/dashboardv1`：
   - 快径：`cp -r $SRC/web/src/gen`（部署树恒有正本）
   - 慢径：`buf generate --template buf.gen.web.yaml`（镜像内同法）
2. dist 快照语义注明：拷贝件仅供闸/dev，生产前端只走 deploy_image.sh（Dockerfile 内 vite 重建）；
   既拷 dist 时打一行快照警示。
3. KB 条目 `worktree-provision-missing-web-gen.md` 补「根修=T-0229」记。

## 验收锚

1. 干净 worktree 跑 provision 后三件齐（`web/src/gen/dashboardv1/core_pb.ts` 在册）
2. check_all 全链绿；既有配供路径（gen/dist 已在→免补）不回归
3. KB 条目根修注在册

---

## 验收注（`[角色:决策]` @ 2026-10-28）

抢修闭环：①干净 worktree provision 实证三件齐（web/src/gen/dashboardv1/core_pb.ts 在册）②check_all 全链绿（t0229 worktree 戳）③KB 根修记随修件同 commit（D-585）。dist 快照=闸/dev 用途明示——生产混装面由 deploy_image.sh 出处闸+镜像内重建兜住，本修断的是 gen 缺件主线。
