# servemux-pattern-duplicate-panic — mux 同 pattern 双注册=boot panic

## 事故
T-0068 在 `mountUI`（service.go）注册 `mux.HandleFunc("/")` 时，未核 main.go 已有
`/` 探活件（返 "ok"）。Go 1.22+ ServeMux 冲突 pattern → boot panic → 生产全瘫。
更阴的是竞态时窗：竞态重建出的容器跑的是无 panic 的中间二进制，掩盖病灶数分钟。

## 判据
- **挂任何 `/`、`*`、裸前缀 pattern 前**：先 grep 全仓 `mux.Handle`/`HandleFunc` 同
  pattern 注册面；mux 注册是全局命名空间，不是文件局部事。
- **验收 anchor**：改 mux 挂载面的票，验收必须含「进程真起得来」——光 curl 通
  （旧进程还在服）不算数，要看新容器 boot 日志零 panic。
- 探活端点有家可归：healthcheck 认 200 即可，`/` 让位活页后探活件该删则删。

## 关联坑
- worktree-web-dist-embed-gate（前端件 embed 面同族：本地有≠镜像有）
