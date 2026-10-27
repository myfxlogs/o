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

## 状态

- [x] 立单
- [ ] 施工
- [ ] 验收投产

## 结果回填

- universe 过滤器 HasSuffix("-USDT")→Contains("-USDT-") 两处同修
  （cmd/collector/main.go:130+cmd/core/universe.go:59——期货 instId
  日期尾形笔误级 bug 根修；datedFutRe mark-price 订阅分支零动）。
