---
schema_version: 2
id: dev-shell-rtk-wrap-coreutils-format-drift
kind: pitfall
risk: low
status: verified
keywords: [rtk, ls, ps, find, 本机自测, 输出格式, devin-会话]
scope: [headless, script-test, dev-shell]
summary: devin 会话内 ls/ps/find 被 rtk 包壳改输出格式（ls 出 size 列、-t 失效）——本机自测脚本逻辑假阴性，生产 systemd 径无壳正常；自测解析 ls 输出=伪证
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: ticket
    ref: T-0267
---

## devin 会话 rtk 包壳 coreutils——脚本本机自测输出失真

- **实证**（T-0267 件C 调试）：`ls -t` 在 devin exec 内经 rtk 包裹——输出
  形如 `name  2.5K`（附 size 列）且排序语义漂，脚本 `for f in $(ls -t …)`
  全空；同脚本裸 bash（生产 systemd 径）正确。pgrep -f 的 argv 也被 rtk
  前缀包（`bash -c rtk ps …`）——让位协议三相核实里锚 argv[0]/[2] 定界
  的既有写法已防，自写 pgrep 解析需注意。
- **教训**：devin 会话内自测「依赖 coreutils 输出格式」的脚本=伪证径——
  要么用 bash 内建（`${f##*/}`/`-nt`/glob）写脚本，要么显式 `/bin/ls`
  复测；勿凭会话内输出下「脚本错」结论反向亦然（生产对会话）。
- **收口**：T-0267 status_block 改纯 glob+bash 内建实现（零 ls/stat fork，
  16k 文件 0.5s）。
