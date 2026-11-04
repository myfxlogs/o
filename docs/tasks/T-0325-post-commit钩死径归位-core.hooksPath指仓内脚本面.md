# T-0325 post-commit 钩死径归位——core.hooksPath 指仓内脚本面

| 字段 | 值 |
|---|---|
| 状态 | 草案（让位 T-0326 部署链——COS 令 11-04：部署票派毕即时重批，件本已审过） |
| 立项 | `[角色:COS]` devin @ 2026-11-04（决策席 T-0324 验收注记交办——其注记称「已立草案」实未落盘，COS 补立；根因实证：坑条目 `core-hookspath-dead-install-face` @f958266d 同案在档） |
| 领取 | （空=未领） |
| 优先级 | 中高——自报行欠账类故障的**总根因**，每漏一次人肉代录一次 |

## 病灶（实证在案）

`git config core.hooksPath` 指向 `scripts/hooks`（仓内目录）→ `.git/hooks/`
里安装的钩件**自始永不触发**。此前所有「收口自报行没落账」谜案
（T-0317 返修 a286cea7、T-0323 返修 86d82229——均曾归因于「钩匹配词
不含返修字样」并加宽匹配）**真凶是钩根本没跑**：匹配词修不修都无
输出。O-Dev 10:42–43 的三行自报实测为手动调用，`absence-detector`
坑条目的成因面自此完整。

## 钦定落形（勿再裁方向）

1. **钩件入仓**：`scripts/hooks/post-commit`（版本化、随仓分发）——
   内含现行逻辑：subject 匹配 `收口|返修|移交` → `report.sh 收口
   <T-号>`（T 号从 subject `[T-NNNN]` 提取；无 T 号静默退）。
2. **worktree 覆盖**：`provision_worktree.sh` 确认新工作树继承
   `core.hooksPath`（config 是仓级非 worktree 级——验证并注档，
   若 worktree 不继承则在 provision 脚本里补 `git config` 行）。
3. **部署树 .git/hooks 死件清场**：`/opt/o/.git/hooks/post-commit`
   留或删须一档记死（建议删——死径误导后人再装一次同样的钩）。
4. **实证随收口**：收口前干跑——在分支提一个含 `[T-0325]` 的测试
   commit（可空 commit），`ops_event` 落收口行即证钩活。

## 验收锚

- [ ] `scripts/hooks/post-commit` 在仓+可执行位
- [ ] `core.hooksPath` 指向验证实录（`git config --get` 输出贴票）
- [ ] worktree 继承性结论落票面（继承/补配 二选一实测文）
- [ ] 空 commit 干跑→ops_event 收口行实录（行 id 引票）
- [ ] 死径坑条目回改点：条目内「待归一」段改引本票完成态

## 范围外

- report.sh 匹配词/事件族——已宽化在案，勿再动
- 其他钩（pre-commit 门禁系）归属本票不动——若查明亦死径，另立票

## 查档

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 席注记称「已立 T-0325 草案」实未落盘 | 审计注记虚报产物——机制首例 | COS 补立本票；记档不追责（注记声称 vs 落盘实的差异，席位协议尚无「声称即落」校验） |
