# 体制读数模板（工单O-32603——唤知会话 regime:due 触发专用）

本拍唤起 trigger 携 `regime:due`=体制读数到期。你只产**一行结构化
特征行**——读数不接任何执行/闸/策略消费（消费侧另案，证伪满窗后
才谈接线）。禁自由文本判词外溢，标签走闭集。

## 输入面（先跑完再判——evid 锚=本次实读水位）

```bash
# 1) 流动性体制原料：md_funding 近 7d 结算率分布（正=多付空收拥挤,
#    负=反向;离散度=体制温度)
psql "$FIN_DSN" -qAt -c "SELECT inst_id, count(*),
  round(avg(funding_rate)::numeric,6) avg_r,
  round(stddev(funding_rate)::numeric,6) sd_r,
  round(avg(abs(funding_rate))::numeric,6) abs_r,
  max(ts)::timestamp(0) last_ts
FROM md_funding WHERE venue='okx' AND ts>now()-interval '7 days'
  AND inst_id IN ('BTC-USDT-SWAP','ETH-USDT-SWAP')
GROUP BY 1 ORDER BY 1"

# 2) 趋势/波动原料：md_klines 1D 确认棒（宇宙横切口径同 regime_beat）
psql "$FIN_DSN" -qAt -c "WITH b AS (
  SELECT inst_id, ts, c, row_number() OVER w rn, lag(c) OVER w prev
  FROM md_klines
  WHERE venue='okx' AND bar='1D' AND confirm=true AND inst_id LIKE '%-SWAP'
  WINDOW w AS (PARTITION BY inst_id ORDER BY ts DESC)
), s AS (
  SELECT inst_id,
    stddev_samp(ln(c/prev)) FILTER (WHERE rn BETWEEN 2 AND 21)*sqrt(365) vol,
    ln(max(c) FILTER (WHERE rn=1)/max(c) FILTER (WHERE rn=15)) mom
  FROM b WHERE rn<=21 GROUP BY inst_id HAVING count(*)>=21)
SELECT count(*), round(percentile_cont(0.5) WITHIN GROUP (ORDER BY vol)::numeric,3) med_vol,
  round(percentile_cont(0.5) WITHIN GROUP (ORDER BY mom)::numeric,4) med_mom,
  round(avg((mom>0)::int)::numeric,3) breadth, max(mom>0.03)::int up_n,
  max(mom<-0.03)::int down_n FROM s"

# 3) 市场面事件 24h（体制判读的事件锚——崩盘/解禁/宏观冲击在账即读）
psql "$FIN_DSN" -qAt -c "SELECT event, count(*) FROM ops_event
WHERE ts>now()-interval '24 hours'
  AND event IN ('升件','处置','部署','量表') GROUP BY 1 ORDER BY 2 DESC LIMIT 8"
```

## 判读判据（机械可复核——禁凭感觉）

- **liq_\***（流动性体制）：看 funding abs_r 中位与离散——
  abs_r 高∧sd_r 高=拥挤加杠杆市→`liq_tight`；abs_r 低且稳→`liq_ease`；
  其余→`liq_neutral`。
- **risk_\***（方向风险态）：看 med_mom/breadth/事件面——
  breadth>0.6∧mom 正向=趋势上行风险开→`risk_on`；breadth<0.4∧mom
  负向=下行/避险→`risk_off`；breadth 0.4–0.6∧|mom| 小=无主导方向
  →`risk_thin`。
- **conf**：三源输入互相印证=high；两源一致一源缺/陈=mid；输入
  缺腿（如 funding 表空窗）或互相矛盾=low。
- **horizon**：体制快变量（事件冲击/急 funding 漂移）=24h；慢变量
  （趋势/广度格局）=7d。拿不准取 24h——对拍窗短=证伪快。

## 产出契约（唯一合法写径——先查重再落行）

```sql
-- 标签闭集={liq_ease,liq_neutral,liq_tight}×{risk_on,risk_off,risk_thin}
-- 标签形=liq_<x>+risk_<y>（如 liq_ease+risk_on）；闭集外=违约不落行。
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '唤知机','量表','regime_digest',
 'regime=<liq_*+risk_*> conf=<low|mid|high> horizon=<24h|7d> evid=fp:<8位>;<锚列表>'
WHERE NOT EXISTS (
  SELECT 1 FROM ops_event x WHERE x.event='量表' AND x.ticket='regime_digest'
    AND x.note LIKE '%regime=<同标签>%conf=<同置信>%'
    AND x.ts>now()-interval '24 hours');
```

- `evid` 必载：`fp:<regime_due 标件 fp 前8位>`（对拍回指唤起因）+
  实读锚（如 `md_funding:7d@<last_ts>;md_klines:1D@<last_bar>;ops_event:24h#<id>`）。
- **去重窗=24h**：同标签+同置信 24h 内已落行→本拍不产（查询返回
  空=正常收工，非失败）。标签或置信任一变=新读数可落。
- 龄界自觉：你只产当下读数；horizon 满后由 regimedigest.sh 对拍段
  机械评分（regime_score 行）——你不写 score 行、不改旧读数。
