# T-0467 部署收敛面根修——推已绿 sha + 部署树 commit 时闸

- 状态：完成（2026-11-08 外审裁过归宗——外派件走外审径，实裁=外审席）

立项：2026-11-08 业主令「用外部手术刀根修内联问题，拒绝内耗」+「镜面任务完成后统一部署」形制钦定。**外派件**——T-0409/0460 同径，归宗裁=外审席。

## 病灶（11-08 全天实证）

收敛口四病齐发，数学死局：

- **跑步机**：戳绑起跑 SHA、推要 HEAD 持戳——gate 20-40min vs commit 分钟级，服务<到达=队永胀；当日废戳 4 次。有交互靠人肉等安静窗侥幸，全无头=无限等待。
- **违例进历史**：2 个施工前缀 commit 直落 master 入 first-parent，§9 闸在 push 才检=检出即晚期，须 rebase 手术摘除（已施，27 件树逐字节无损）。
- **票号毒源**：T-9999 销号墓碑占远端最大号，0462+ 新票全塞（已改名脱域）。
- **环境放大器**：盘 100%/孤儿锁/GOCACHE 互逐——环境面另票 T-0466 已批，本票不重复。

## 钦定面（业主令，勿再裁方向）

① **推已绿 sha 非 HEAD**——`git push origin <stamped_sha>:master`：戳验的本是「该 SHA 的树测过」，推它=推已验证物；跑期间新落 commit 归下趟不冻不废。机制上 pre-push `stamp_ok` 读被推 sha 即原生支持。
② **统一部署镜面制**——「随到随推」改「镜面发窗」：取最近绿戳 SHA 推 + deploy_image 按该 sha 发；上游 merge 连续进食不堵，下游定期出窗。
③ **部署树 commit 时闸**——/opt/o pre-commit 拒「非 merge 非授权席」直落 master（§9 前移：违例进不了历史则永无手术）；dialogue/票件写径引去会话己支随 merge 入。

## 裁填面

- 镜面发窗形制：周期（timer/事件驱？）+「取最近绿戳」的选择口径 + 与既有 git_push.sh/master_push_converge.sh 的并存关系（是替还是包）
- commit 时闸白名单集：哪些席/词形可在部署树直 commit（决策 merge 件天然正当？COS 台账件？判据集须票面钦定防回归 T-0370 语义）
- 与「绿了才推」业主令语义核对：推已绿 sha 不降格（验证在前，推在后）

## 判据

- 并发 commit 期间，`git_push <已绿sha>` 可推——HEAD 移动不废戳（桩测：跑中落 docs commit→推已绿 sha 过闸）
- 施工前缀直 commit 部署树 master→pre-commit 即拒，进不了历史
- 全 check_all/既有三闸语义不破；新径弃置可回旧径

## 结果回填（施工 2026-11-08）

### 实现摘要（三钦定面全落）

- **件①推已绿 sha**：机制核实=pre-push `stamp_ok` 本就读被推
  local_sha（非 HEAD）——原生支持 `git_push.sh origin <sha>:master`；
  git_push.sh 头注补用法指针。桩测实证：戳在 P1、HEAD 走至 P2，
  推 `P1:t/probe` 过闸 rc=0，无戳 HEAD 推拒 rc=1。
- **件②镜面发窗**：新 `scripts/mirror_push.sh`——部署树扫
  `.git/check-green-*` mtime 新→旧，首个三断言全过者推
  `<sha>:master`：①戳龄≤21600s（与 stamp_ok 同 6h 窗）②未被
  origin/master 含（已合入跳过）③origin/master 是其祖先
  （ff 可推，非快进跳过归合流闸）。全灭=镜面空窗合法零动作。
  **并存=包非替**：复用 git_push.sh 重试包装；
  master_push_converge.sh pre-push 合流闸原位不动（镜面推的
  local_sha 恒为 origin/master 后代，C0 判定天然直通）。
  发窗形制裁填=systemd timer `deploy/o-mirror-push.timer`
  （15min 拍——check_all 20-40min 节奏下足够紧凑，空窗零成本；
  单元件落仓不启用，启用=部署面动作归外审/部署径）。
  部署半新 `scripts/deploy_at_sha.sh <sha>`——绿戳在案∧已并
  origin/master 双断言→临时 worktree 检出该 sha→bootstrap→
  跑该 sha 自带 deploy_image.sh（部署物=该 sha 树逐字节，
  出处闸/SG 锁/SHA tag/健康闸全继承）。
- **件③部署树 commit 时闸**：新 `scripts/check_deploy_commit_seat.sh`
  判件+`scripts/hooks/commit-msg` 接线（判据段置于 fix 信号
  早退之前保证全 subject 覆盖）。判法=root realpath=/opt/o
  ∧branch=master∧无 MERGE_HEAD→subject 首词位 `[角色:施工]`
  即拒。判别域=施工前缀负向列（裁填：T-0370 语义一字不动——
  当年闸只禁施工前缀；origin/master first-parent 500 件实证
  决策/COS/无标票立/外审/管理/讨论/秘书直落皆合法径在案，
  施工=唯一违例类；判别域只窄不宽，他席扩禁需另票实证）。

### 裁填面回填

- **发窗形制**：周期制=timer（非事件驱——绿戳无事件源，
  check_all 戳落是文件写非事件；15min 拍 vs gate 20-40min
  节奏收敛足够）；「取最近绿戳」口径=mtime 新→旧首个三断言
  全过者（非简单最新戳——陈戳/已合入/分叉自动落次，
  防「推了个已在远端的 sha」空转与「推分叉件」非快进死锁）。
- **commit 时闸白名单集**：负向列制——仅禁 `[角色:施工]`
  前缀（词形=subject 首词位锚定，防正文提及误伤）；merge
  在途（MERGE_HEAD）豁免=合支正当径天然合法；无标 commit
  放行=DM/决策票立流实证常态（T-0440 系无标立票件）；
  `[角色:决策]`/`[角色:COS]`/`[角色:外审]`/`[角色:管理]` 等
  他席标直落放行=既有合法径不收窄。
- **「绿了才推」语义核对**：不降格——镜面只推持新鲜绿戳之
  sha（戳龄≤6h 与 pre-push 同窗双保险），推的物=验证过的物。

### 偏离说明（裁填层两处，判据语义等价达成）

- 判据字面「pre-commit 即拒」→ 实落 **commit-msg**：subject
  词形在 git hook 序中 pre-commit 阶段物理不可见
  （pre-commit→prepare-commit-msg→commit-msg→post-commit），
  commit-msg=commit 时链最后可见 subject 的闸点，拒=commit
  中止=进不了历史，与「即拒」语义等价；且与既有
  check-deploy-tree.sh（pre-commit 内容面闸：master 非 docs/
  staged 即拒）形成「内容闸前置+席位闸后置」双层，§9 前移
  语义完整。判件独立成脚本（expected_root 第四参仅桩测覆写，
  hook 调用恒缺省=/opt/o 无 env 绕过径）。
- 桩测 C5 以 expected_root=scratch 实证「非部署树直通」——
  /opt/o realpath 本体不可在 scratch 伪造，判件 realpath 判据
  与 check-deploy-tree.sh 同构已在生产在案。

### 测试证据

- `scripts/meta/tests/t0467_deploy_converge_test.sh` 13 场景
  全绿：A 组镜面选择五态（最新戳中选/戳陈>6h 落次/全合入空窗/
  分叉戳跳过/支名无关最新戳中选）+B 组戳绑被推 sha 两态
  （HEAD 移动后推旧绿戳过闸/无戳拒）+C 组部署树闸六态
  （施工拒/决策放/无标放/非 master 直通/非部署树直通/
  MERGE_HEAD 豁免）。
- 过程实录：首版判件 case 模式 `"\[角色:施工\]"*` 引号内
  反斜杠被当字面量致永不命中（红桩实证 C1）→改无引号
  `\[角色:施工\]*` 修复——引号内 `\[` 匹配的是「反斜杠+[」，
  glob 转义须落在未引号段。
- check_all 绿戳：见收口 commit（戳=check-green-<HEAD> 对口）。

### 范围外确认

- 未部署（deploy_at_sha.sh 仅落件——「串行勿部署」钦定）；
  o-mirror-push.timer/service 落仓未 enable（启用=部署面）。
- 未合并；未动 deploy_window_due.sh 两面判据/git_push.sh
  重试语义/master_push_converge.sh 合流语义/check_all 档制/
  pre-push 既有三闸（origin-only/戳龄/master 形）一字不动。
- T-0370 push 时 first-parent 闸原位保留=纵深防御不撤。
- KB 条目 deploy-stamp-sha-not-head-treadmill 同 commit 入册
  +kbgen 索引重建。

## 裁记

- [外审裁:T-0467] 过 @7f3190256 | 断言对拍 3/3：①推已绿
  sha 径=支 wt check_all 独立复跑全链绿（mode=full，戳
  @7de44489a）+master 径真 pre-push 干跑实证（戳在 P3、HEAD
  走至 c4，推 P3:master 过戳闸止于 repo_root 沙盒界——戳绑被
  推 sha 非 HEAD 钉死；无戳 HEAD 对照死戳闸）+hook stdin 实证
  local_ref=<sha>本体不触「*:* 禁映射形」判+C0 合流闸对 ff
  sha 天然直通（rebase 副作用仅非快进竞态时——钦定合流语义）；
  ②commit 时闸=判件桩测独立复跑 13 场景全绿+/opt/o 生产树
  commit-msg e2e 实证（[角色:施工] subj on master rc=1 拒——
  §9 前移成立；决策标/无标/非 master/非部署树/MERGE_HEAD 直通
  六态齐）+D-651 相容（内容闸 pre-commit 前置+席位闸 commit-msg
  后置双层互补，subject 词形 pre-commit 物理不可见——偏离说明
  语义等价成立）；③既有闸零破=pre-push 三闸/git_push 重试/
  converge/first-parent/check_all 档制一字未动（diff 核读），
  mirror_push 包非替可弃回旧径 | 戳 7f3190256（merge-sha
  check-green，/opt/o 复跑 mode=full 绿）| 部署态=闸件检出即
  效（commit-msg 闸随 merge 已活；o-mirror-push.timer 落仓未
  enable——启用=部署面动作归部署径钦定，deploy_at_sha.sh 同；
  「串行勿部署」在案）| 旁证：全角冒号 subject（[角色：施工]
  形）可绕 commit-msg 与 check_master_direct 同源前缀盲区——
  生态一致非本票回归，全域前缀闸统一加宽归另票
