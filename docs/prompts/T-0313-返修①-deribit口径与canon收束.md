[角色:施工] T-0313 返修①——deribit 年化口径错配+canon M 尾霰弹收束

## 任务单
/opt/o/docs/tasks/T-0313-venue-scout漏斗补齐-六候选adapter与canon方言扩.md
原分支 t/0313-scout-adapters 续提，不切新支。

## 驳回件

**B1（硬伤）deribit rate/interval 错配——年化低估 8×**

collect6.go collectDeribit：`rate: f2f(t.Result.F8H / 8)`（8h 累积率折
等效 1h 率）但 `interval` 未填=0→metrics settles() 走 config 回退 3/天
→年化=(f8h/8)×3×365=f8h×136.9，真值 f8h×3×365=f8h×1095——**8 倍低估**，
deribit 榜永失真。

修=二选一钉死口径配对：rate 原样 `f2f(t.Result.F8H)`+`interval:8`（与
他家 8h 率语义同构——**首选**），或 1h 率配 `interval:1`。改注释同口径。

**B2（地雷）canon M 尾裸剥越界**

series.go canon()：`strings.HasSuffix(s,"M")` 在 quote 剥之前裸剥 M 尾
——带 quote 的符号安全（ATOMUSDT 尾 T 不触），但**无 quote 的 M 尾符号**
会被误剥：`ATOM-PERP`→剥 PERP→`ATOM`→剥 M→`ATO`（deribit/dydx 裸名
径可及——PERPETUAL/USDT 尾缺位时真炸）。生产 inst 当前全带 USDT 尾故
未现形，属待爆地雷非实弹。

修=M 尾只准在**复合尾**里剥：把 HasSuffix("M") 裸判换成循环尾表
{"USDTM","USDCM","USDM"}（kucoin XBTUSDTM→XBT 路径不变），ATOM/
DYM/BMT 类裸 M 尾符号零触碰。canon 用例补：XBTUSDTM→BTC、ATOM→ATOM、
DYM→DYM、BTC-PERPETUAL→BTC 四条回归钉。

## 验收
- go build ./... + check_migrations + scout 包 go test 全过（含新 canon 用例）
- deribit 单测桩：funding_8h=0.0008 输入→落行 rate=0.0008 interval=8
  （或 rate=0.0001 interval=1 若取第二径）——年化口径实证非注释
- 部署实迹照旧由决策席部署拍后验

## 自审记录
| 编号 | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | B1 修法两径给倾向未钉死 | 去歧义 | 钦定首选 rate 原样+interval=8（与他家同构最少假设） |
| 2 | B2 是地雷非实弹易弱化处理 | 严重度界定 | 明示「未现形但结构必炸」——同一返修内修不延后 |
| 3 | canon 修法的复合尾表是否够全 | 完备性 | 三复合尾覆盖 USD 族全部；非 USD 计价 M 尾不计（现域无此面） |
| 4 | 回归用例清单逐条给 | 硬门 | 四条钉死——ATOM→ATOM 是反地雷锚 |
| 5 | deribit 桩测数值钉死 | 可验锚 | 0.0008→rate/interval 两值断言，非「能跑过」 |
