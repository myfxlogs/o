# T-0460 推程堵塞急修——tolerate 锚补位+check_seat_writes 判别粒度修

- 状态：已批（CTO 自立票 2026-11-08——#3704 升件/#3711 CEO 处置移交 CTO 急修件）
- 领取：常径待派
- 优先级：P1（全管线推送面堵塞——pre-push 戳闸不可达，43 commit 滞留本地含 T-0145 已裁合件）
- 收口移交归：决策席
- 立项：check_all fail-fast 于 tolerate-report（12+ 处新 `|| true` 未锚未册），
  seat-writes 闸判别 pattern `*"claude-glm@local"*` 命中全席（本机各席同署
  claude-glm@local——[角色:施工]/[角色:COS] 合法 commit 全被错标为决策席
  越权件，判别名义=「决策席会话产出」实证过宽）。

## 病灶（亲证在案）

1. **tolerate-report 红**：`emit_hits` BAD 清单——
   `check_seat_writes.sh` 3 处、`ceo_tick.sh` 6 处、`dm_autoapprove.sh` 1 处、
   `dm_suspend.sh` 1 处、`dm_tick.sh` 3 处 `|| true`/`|| :` 新位无
   `# tolerate:` 行内锚且未入 docs/容错位清单.md registry。
2. **seat-writes 判别过宽**：`case "$subj|$author"` 的 author 兜底项
   `*"claude-glm@local"*"`——本机施工/COS/决策全席同署，凡触非白名单件
   commit 一律 FAIL；merge 检查同理（支侧施工 commit 被计入 allmine）。
   立法名义只盖「无头决策席会话产出」——他席标 commit 错标=伪阳性，
   逼使一切推送走 O_SEAT_WRITE_ALLOW 毯式豁免（豁免常态化=闸无信号）。

## 范围（钦定）

1. BAD 清单各行补 `# tolerate: <原因>` 行内锚——锚因如实逐条写
   「败了谁知道/为何可吞」，禁糊词（对照 docs/容错位清单.md 已册位同
   类口径）；确属应入册者走 `--emit-registry` 机生径刷新 registry
   机读块（禁手编）。
2. check_seat_writes.sh 判别粒度修：author 兜底收窄为
   **「subj 无 `角色:` 标且 author=claude-glm@local」**才计决策席产出；
   `[角色:决策]`/`[角色:CTO]` 标判别域一字不动（T-0145 实证 catch 必保
   ——[角色:决策] 标 commit 触非白名单仍 FAIL，merge 支侧全决策席产
   自合仍 FAIL）；`[角色:施工]`/`[角色:COS]`/其他角色标 commit 出判别集。
3. 桩测钉死判别三类断言：施工标触码面=PASS、决策标触码面=FAIL、
   无标 claude-glm 触码面=FAIL（CHECK_SEAT_RANGE 覆盖径现成）。

## 验收锚

- check_all 全谱绿——tolerate-report+seat-writes 两节过（桩测断言
  修前红修后绿——先红桩律：红因=现 BAD 清单+判别伪阳性在案）；
- 判别桩测三断言绿；
- 本件收口即推送面解锁实证（绿戳落 HEAD——存量 [角色:决策] 标违例
  件推程按 #3569 令 O_SEAT_WRITE_ALLOW 豁免径另行，不属本票）。

## 红线

- 只动 BAD 清单行锚+check_seat_writes.sh 判别段——registry 机读块
  禁手编、判别域只窄不宽、既有 anchor/豁免径语义零动；
- 不改 check_all 步序/其他闸件；不碰 app_config。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
