# T-0189 宇宙过滤器吞掉到期期货——basis_ann 结构性零行根治

> 立单：决策席 2026-10-27（「已知缺陷也需要修理」业主令——basis_ann
> 黄注挂账到今根查）
> 域：`cmd/collector/main.go` universe()（+cmd/core/universe.go 同语义
> 复制件核对）、dev collector 订阅宇宙
> 量级：微件（一行过滤器修+验收实证）

## 病灶实证（决策席取证在案）

`basis_ann` 自 T-0164 起恒 0 行被记「已知缺席」——根查非市场缺席，
是**订阅宇宙源头吞标的**：

```
md_mark_index 507 行：channel=mark-price|index-tickers 全 -SWAP，
inst_id ~ '-[0-9]{6}$'（到期期货形）= 0 条

cmd/collector/main.go:130
  i.State=="live" && i.Alias=="quarter" && strings.HasSuffix(i.InstID,"-USDT")
                                                  ^^^^^^^^^^^^^^^^^^^^^^^^
  到期期货 instId 形 = BTC-USDT-251226（尾缀=日期）
  HasSuffix("-USDT") 永假 → 季度期货整族从未进宇宙
  → 无订阅 → 无 mark-price 行 → basisAnn JOIN 恒空集
```

**过滤器笔误级 bug**——判 `Contains(i.InstID,"-USDT-")` 才是期货
USDT 保证金键域（形如 X-USDT-YYMMDD）。

## 修法钦定

`universe()` 季度期货过滤改 `strings.Contains(i.InstID, "-USDT-")`
（或 `-USDT-` 中缀判）。`cmd/core/universe.go` 系同语义 SG 瘦身面
复制件——**同查同修**（SG 虽只活跃集订阅，universe 函数同误则同修，
防语义漂移）。

注意副锚：期货行进场后 `activeSet()`/tickers 面不受影响（tickers
只订活跃集∩宇宙）；`subscribeArgsFull` 的 `datedFutRe` 分支本就
为期货 mark-price 预留（订阅器端零改）。

## 验收锚

1. instruments FUTURES live+quarter 的 -USDT- 形标的进宇宙集
   （日志 syms 计数应涨——现 1014 为纯 SWAP+现货对）；
2. `md_mark_index channel='mark-price' AND inst_id ~ '-[0-9]{6}$'`
   出现行（季度期货 mark 帧流入）；
3. `basis_ann` 拍出行数 >0（`[sigdist] basis_ann 拍：N 行`，N>0）；
4. System 页「蒸馏 basis_ann」由黄转绿；
5. `cmd/core/universe.go` 同点核对修或注记豁免（SG 面宇宙仅
   注册径不订阅——若同误同改）。

## 决策席验收返修（2026-10-27——票面判据被 API 实证证伪）

**我立单时的 `-USDT-` 判据错了**——venue 实测：

```
/api/v5/public/instruments?instType=FUTURES  live quarter 全 6 枚:
  BTC-USD-261225 / ETH-USD-261225 …… 全是 -USD- 币本位
  -USDT- 保证金到期期货 = 0 枚（OKX 不挂此形）
```

施工照票面改 `-USDT-` 中缀判=仍空集（合入无害但证不了 basis_ann）。
**二次裁钦定**：

1. 过滤改 `datedFutRe` 形判（`-\d{6}$` 尾）或 `Contains("-USD-")`——
   凡 quarter 到期期货不论保证金种全收（6 枚实证）；
2. **指数键域配套**：basisAnn JOIN 剥日期得 `BTC-USD` 对
   index-tickers `inst_id='BTC-USD'`——universe 须把期货 uly
   剥离键（`BTC-USD-261225`→`BTC-USD`）一并入集，index-tickers
   订阅面才会覆盖 USD 指数行（现只有 -USDT 指数）。漏此步=
   期货 mark 到了仍 JOIN 不上；
3. `cmd/core/universe.go` 同修（同两处语义）。

验收锚改判：`md_mark_index` 出现 `*-USD-*` 到期期货 mark 行+
`BTC-USD` 形 index 行，`basis_ann` 拍行数 >0 方算绿。

## 决策席二次返修（d7c0d84 实证——两层断点俱在）

**断点 A·剥离键取错源**：施工用 `datedRe.ReplaceAllString(instId,"")`
剥 instId——`BTC-USD_UM-261225`（OKX _UM 产品线与裸形并存，uly 均
=`BTC-USD`）剥出 `BTC-USD_UM` 非法指数名，index-tickers 订阅被
60018 拒（日志实证）。**钦定改法**：instruments 载荷自带 `uly`
字段=钦定指数键——`out = append(out, i.Uly)`，禁再 instId 剥串。

**断点 B·帧径 USDT 硬闸拦收**：`isUSDTInst`（parse.go:56
`Contains("-USDT")`）在 handle.go:32 把所有非 -USDT 键域帧入队前
丢弃——**订阅就算成功，期货/USD 指数帧到不了库**（零行实证）。
钦定改法：`isUSDTInst` 加豁免——`-\d{6}$` 到期期货形 + `-USD$`
指数键形放行（注释钉 T-0189：业主「杂币不采」令禁的是杂计价币
面，USD 指数/币本位期货系 basis 域钦定数据源，非杂币）。

- [x] 立单
- [x] 施工（一巡：-USDT- 判——判据错空集，返修）
- [x] 施工（二巡：datedRe 尾判——期货入集实证 syms 1014→1030，
      但三层断点仍未通，返修③）
- [ ] 施工（三巡：uly 字段键+USDT 闸豁免）
- [ ] 验收投产

## 结果回填

- universe 过滤器 HasSuffix("-USDT")→Contains("-USDT-") 两处同修
  （cmd/collector/main.go:130+cmd/core/universe.go:59——期货 instId
  日期尾形笔误级 bug 根修；datedFutRe mark-price 订阅分支零动）。

## 状态

审计通过

> 审计席：headless @ 2026-10-30

## 判决注记（无头审计——收口 c640660+99204d2 逐锚对账，返修改判锚口径）

- 锚① **过**：在役件 vcs.revision=aa21046（`merge-base --is-ancestor`
  双核——c640660/99204d2 均真祖先）；journalctl o-collecto 实拍
  `[okxws] subscribed syms=1034`（10-27 22:29/10-28 13:16 两拍——
  基数 1014 实涨）；码面 datedRe `-\d{6}$` 尾判+`i.Uly` 追加两处
  在 origin/master 逐字核。
- 锚② **过**（改判口径）：SG 库只读实拉（ROSTER 隧道 :5544）——
  md_mark_index 到期期货 mark 行 6 枚在册（BTC-USD-261225/
  BTC-USD_UM-261225/ETH-USD-261225/ETH-USD_UM-261225/
  SOL-USD_UM-261225/XAU-USD_UM-261225，帧 ts 至 10-30 09:55Z 仍鲜）；
  `*-USD` 指数键行 4 枚（BTC-USD/ETH-USD/SOL-USD/XAU-USD）——
  「`*-USD-*` 到期期货 mark 行+`BTC-USD` 形 index 行」改判字面全中。
- 锚③ **过**：`[sigdist] basis_ann 拍：2 行` 60s 拍连打 10-27 22:00→
  10-28 14:18（服务停拍止）；md_signal@SG 现存 basis_ann 两键
  （BTC-USD-261225=565.79bps/ETH-USD-261225=400.45bps，
  末写 10-30 02:55Z——流延 ~28h 跨部署）N=2>0。
- 锚④ **过**（机理级）：瓦片判据=同库 `max(ts) md_signal
  kind='basis_ann'` 鲜度（ops_face.go:47-49，warn 10min）——行在
  且鲜→服务端派生绿；黄态本=零行 zeroState=1 语义（T-0241 已改
  zeroState=0），行实到即灭。附注：写判决时该键行龄 ~7h（采集面
  02:55Z 后停拍）+本机 o-collecto 自 10-28 14:18 停役——现红=
  管道停运运维面，非本票病灶；锚义「数到转绿」在证窗内达成。
- 锚⑤ **过**：cmd/core/universe.go 同点同修（datedRe 尾判+i.Uly
  追加+T-0189 钉注）——diff 逐字同构零漂移。
- 钦定修法对账：断点A uly 载荷字段键（`out=append(out,i.Uly)`，
  _UM 剥串错源根治，钉注在码）；断点B isUSDTInst 豁免
  （`-USD$` 尾+datedFutRe——「杂币不采」令辨析注释钉 T-0189）；
  sigdist datedFutRe 宽判 `^([A-Z0-9]+-[A-Z]+)-([0-9]{6})$`
  （99204d2 真落库——蒸发回归坑条目 acceptance-hotfix-uncommitted-
  evaporated 同 commit 在册）。
- 残留注记（报管理席，不挡锚）：`_UM` 产品线 mark 行可收不可
  JOIN——basis.go SQL 侧 `regexp_replace('-[0-9]{6}$','')` 剥得
  `BTC-USD_UM` 非法指数键（指数键=uly=`BTC-USD`），6 枚 quarter
  仅 2 枚裸形（BTC/ETH-USD-261225）产 signal，4 枚（_UM+SOL/XAU）
  行在而 JOIN 空——覆盖面缺口建议立微件：JOIN 键改 uly 映射或
  补剥 `_UM` 中缀。
- 桩面核对：parse_test/ghost_test 用例与新豁免语义相容
  （`BTC-USDC-SWAP` 仍拒、幽灵件 valid 闸兜住）；订阅分派
  datedFutRe→mark-price/uly 键→index-tickers 逐字合钦定。
