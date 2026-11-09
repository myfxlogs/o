---
schema_version: 2
id: checkall-branch-surface-scoped-vs-full
kind: lesson
risk: medium
status: verified
keywords: [check_all, scoped, 定向谱, 全谱, merge-base, 分支面, 算力浪费]
scope: [ci, check_all, 分支纪律, 测试谱分级]
summary: check_all 测谱=merge-base→HEAD 全面判级——大杂面支/清单外件(二进制等)强制回退全谱；施工支保票粒度纯面=自动命中 scoped 定向谱
first_seen: 2026-11-09
last_verified: 2026-11-09
skills: []
evidence_refs:
  - type: doc
    ref: scripts/check_all.sh:87-104
  - type: doc
    ref: docs/tasks/T-0672-草案进水口号耗根修-物料化发号+层扫账面类类级节流.md
---

## check_all 测谱由分支变更面自动判级——支面不纯=全谱回退=白白烧算力

- **业主令（2026-11-09）**：后续施工不得再无谓跑全谱——时间和算力
  不该被大杂面分支烧掉。
- **判级机制（scripts/check_all.sh:104-213）**：`MODE` 由
  `git diff merge-base(HEAD,origin/master)...HEAD` ∪ 工作树件的
  **全差面**判级：
  - 全 docs/skills/*.md → docs-only 免测；
  - .go 纯面（+scripts/.github 中性件）→ **scoped 定向谱**
    （变更包+反向依赖闭包+恒测册 risk/ledger/trading/guard/lifecycle）；
  - .go+migrations/proto → merge-scoped；
  - **任何清单外件**（二进制、go.mod/sum、web/、Dockerfile、其他）
    → `HAS_OTHER=1` → **full 回退**（fail-safe 不 fail-open）；
  - `CHECK_ALL_FORCE_FULL=1` 无条件全谱；merge-base/命令败
    fail-closed 走全谱。
- **实证坑（2026-11-09）**：
  ① 纪元清批 commit 带 273 件混合面 → 支上后续 commit 全继承
     full（scoped 算 merge-base 全差，不只看最新 commit）；
  ② 误把 `go build` 产物 `ticketdraft` 二进制 commit 进支 →
     仓根裸 ELF=清单外 → 全谱（15MB 还差点进历史，已摘除）。
- **施工纪律（据此条生效）**：
  1. **一票一支、支面纯净**——施工支只带本票码面；清批/台账/
     纪元件独立支走，不与施工码混支。
  2. **`go build` 一律 `-o /tmp` 出仓**或先查 `git status` 再
     commit——产物进暂存=清单外件=全谱触发器（且大文件闸拦）。
  3. 大面票（跨包重构/迁移/proto/web）该全谱就全谱——本条不
     是禁全谱，是禁「支面不洁造成的无谓全谱」。
  4. 戳前可预判：`git diff --name-only $(git merge-base HEAD
     origin/master)...HEAD | grep -vE '^(docs/|skills/|scripts/|\.github/)|\.md$|\.go$|\.sql$'`——
     有输出=必跑全谱，先想值不值得。
- **反例边界**：scoped 谱不含的包零回归保护——恒测册外的新交互
  面若只靠 scoped 验证，merge 后仍由 CI 服务端全谱兜真伪
  （check_all.sh:264 注释钦定），故 scoped 绿≠免全谱审计场景。
