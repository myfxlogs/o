# T-0199 funding_view 血统下游饿断——阶梯序列「末次确认」口径共振修

> 状态：待施工
> 域：评估面/蒸馏语义 · 决策席直派（T-0196 副作用实证 2026-10-27 夜）

## 病灶实证（T-0196 副作用——血统修对了方向、算错了口径）

T-0196 把 funding_view.ts 从 `now()` 改为源行 `recv_ts`——血统方向正确，
但阶梯序列的观测龄语义错位：md_funding 行 `ON CONFLICT DO NOTHING`
（fin_0066 钦定——唯一键挡重拍），行的 recv_ts = **当期结算行首见时**，
同期间后续帧全部 conflict 不更新 recv_ts → 戳随结算间隔老化（实证：
dev 面 min 50min / max 4h50m；结算公告节拍 1h/4h/8h 不等）。

下游共振（已实证）：
- `carry_diff` 输入闸 `funding_view.ts > now()-interval '120 seconds'`
  （internal/research/sigdist/xvenue.go:91）恒败 → carry_diff 行冻结
  1h21m+（dev+SG 双侧同陈）——**T-0196 修复使 carry_diff 全谱饿断**。
- 消费面凡对 funding_view 用 Fresh-120s 闸者同病（fresh 语义被判
  「永久陈行」=按缺席处置——比原 now() 假鲜更偏向保守，属方向性安全
  但功能性失能）。

## 钦定语义裁决（施工照此，可裁实现细节）

funding_view 的「观测龄」钦定 = **末次确认时**（last-confirmed），
非「当期首见时」：费率值在其结算期间内持续有效，「值是否当值」的
正解是「最近一次推送仍在宣告该值」——阶梯序列的 bloodline 记
「最后一次活着确认它的时刻」。

实现面二选一（施工裁，须同 commit 说明取舍）：

**径甲（推荐）**：md_funding 冲突改 bump recv_ts——
`ON CONFLICT DO UPDATE SET recv_ts=EXCLUDED.recv_ts`：recv_ts 转
「末次观测」语义→funding_view 血统=喂活度（帧流在则恒鲜）。
**须解放大效应**：fin_0066 的 DO NOTHING 本意为挡每帧行写——
改 DO UPDATE 则每帧重写行（tombstone 写放大回归），钦定须以
拍内去重/攒批缓冲解：collector 侧 funding 帧先按 (inst,ts) 帧内去重
+攒批 flush（如 10s/批），批内同键只留最新 recv_ts——放大率=
push率/批窗，可接受；批内去重件已有现成构型可循（ticks 攒批同法）。

**径乙（备选）**：funding_view.ts 改取 `max(recv_ts)` over 该 inst
近窗行（如近 1h）——零写放大，但「当值宣告」与「末次确认」语义
仍偏（近窗无新行=喂死判陈）；carry_diff 等消费闸须同步改为
「当值性」判据（meta.next_funding_ts>now()）。改面多点、语义仍偏，
次选。

钦定下限：carry_diff 等下游 Fresh 闸恢复供血；funding_view 行龄
实证回分钟级（feed 活则鲜）；写放大须可计量（攒批面压测或
WAL/行写率实录钉放大量）。

## 验收判据

- funding_view 行龄部署后回落分钟级（结算周期内不再随小时老化）
- carry_diff 行 ts 恢复随拍更新（两行实证）
- 写放大量有实录钉值（批窗/去重率落票面）
- funding 帧停时 funding_view 随陈（喂死如实——反向判据防
  「恒鲜假活」回归）
- 桩锚绿 + 票面无坑注

## 教训记档

T-0196 票面钦定 recv_ts 时未推演「DO NOTHING 下 recv_ts=首见时
随结算周期老化」——阶梯序列（值在期间内恒效）与流序列（每帧新值）
的观测龄语义不同构；验收时 funding_view 17s 鲜为「恰逢新期间公告」
假绿，1h 后才显形。决策席坑注在案（本票首行直派来源），条目化入
工程经验库：「血统戳取源列前，先钉该列语义=首见/末见/事件时」。
