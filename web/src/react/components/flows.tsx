// flows.tsx —— 复盘页面件库（T-0173 棘轮拆分：历史订单卡/流水筛常量）。
// T-0220b：历史订单首页=ev_position 推帧分片订阅（一跳零重拉；载荷粒度
// 5 行/页——原 10 行翻页粒度同票对齐）；翻页视图=unary 参数取（页参数=
// 视图态，非推送面）。
import { Fragment, useEffect, useState } from "react";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { client } from "../api";
import { hydrate, useSlice } from "../store";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton, UtcClock } from "./ui";
import { pbDate } from "./fmt";
import { DIRECTION_CN } from "./labels";
import type { ListClosedPositionsResponse } from "../../gen/dashboardv1/core_pb";

const PAGE = 5;

export function ClosedPositionsCard() {
  const [page, setPage] = useState(1);
  const [fetched, setFetched] = useState<ListClosedPositionsResponse | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [tick, setTick] = useState(0);
  const live = useSlice("closedPositions");
  const home = page === 1; // 首页=推帧分片；翻页=unary 取页

  useEffect(() => {
    if (home) {
      hydrate("closedPositions", () =>
        client.listClosedPositions({ limit: PAGE, offset: 0 })
      );
    }
  }, [home]);
  useEffect(() => {
    if (home) return;
    let alive = true;
    client
      .listClosedPositions({ limit: PAGE, offset: (page - 1) * PAGE })
      .then((d) => {
        if (!alive) return;
        setFetched(d);
        setErr(null);
      })
      .catch((e: unknown) => {
        if (alive) setErr(e instanceof Error ? e.message : String(e));
      });
    return () => {
      alive = false;
    };
  }, [home, page, tick]);

  const c = home
    ? { data: live.data, err: live.err, loading: live.loading }
    : { data: fetched, err, loading: fetched === null && err === null };
  const total = c.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / PAGE));
  const retry = () => {
    if (home) {
      hydrate("closedPositions", () =>
        client.listClosedPositions({ limit: PAGE, offset: 0 })
      );
    } else {
      setTick((t) => t + 1); // 翻页视图重取（key 指纹=home:page:tick）
    }
  };
  const dur = (a: Date, b: Date) => {
    const h = (b.getTime() - a.getTime()) / 3600000;
    if (h < 48) return `${h.toFixed(0)} 时`;
    return `${(h / 24).toFixed(1)} 天`;
  };
  return (
    <Card title="历史订单（已平仓）">
      {c.loading && !c.data ? (
        <Skeleton rows={5} />
      ) : c.err && !c.data ? (
        <ErrorState msg={c.err} onRetry={retry} />
      ) : (c.data?.positions ?? []).length === 0 ? (
        <EmptyState title="暂无已平仓订单" hint="首仓平仓后此处逐行立账" />
      ) : (
        <>
          <table className="rows">
            <thead>
              <tr>
                <th>合约</th>
                <th>开仓</th>
                <th>平仓</th>
                <th>持仓</th>
                <th className="num">结算</th>
              </tr>
            </thead>
            <tbody>
              {(c.data?.positions ?? []).map((p) => (
                <tr key={p.id}>
                  <td>{p.instId}</td>
                  <td className="mono">{(pbDate(p.openedAt)?.toISOString().slice(5, 16).replace("T", " ") ?? "—")}</td>
                  <td className="mono">{(pbDate(p.closedAt)?.toISOString().slice(5, 16).replace("T", " ") ?? "—")}</td>
                  <td className="ink-3">
                    {pbDate(p.openedAt) && pbDate(p.closedAt)
                      ? dur(pbDate(p.openedAt)!, pbDate(p.closedAt)!)
                      : "—"}
                  </td>
                  <td className="num mono">
                    {p.settledAmount !== 0
                      ? `${p.settledAmount.toLocaleString("zh-CN", { maximumFractionDigits: 4 })} ${p.settledCcy}`
                      : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="pager">
            <button className="btn" disabled={page === 1} onClick={() => setPage(page - 1)}>
              上一页
            </button>
            <span className="ink-3">
              {page}/{pages}（{total} 单）
            </span>
            <button className="btn" disabled={page + 1 > pages} onClick={() => setPage(page + 1)}>
              下一页
            </button>
          </div>
        </>
      )}
    </Card>
  );
}


// OpenPositionsCard 在持订单卡（业主令「持仓订单+详细内容」）：open 持仓
// 表+腿明细展开行（DCD 类零腿=「产品仓无订单腿」如实注）+live 待命部署
// 折行（无仓非僵尸）。ev_position/ev_strategy 信号驱动重取零轮询。
export function OpenPositionsCard() {
  const lv = useLive(["ev_position", "ev_strategy"], () =>
    client.listOpenPositions({})
  );
  const [open, setOpen] = useState<Record<number, boolean>>({});
  const [armedOpen, setArmedOpen] = useState(false);
  const ps = lv.data?.positions ?? [];
  const armed = lv.data?.armed ?? [];
  const fmtTs = (t?: Timestamp) =>
    pbDate(t)?.toISOString().slice(5, 16).replace("T", " ") ?? "—";
  return (
    <Card title="持仓订单" extra={<UtcClock />}>
      {lv.loading && !lv.data ? (
        <Skeleton rows={4} />
      ) : lv.err && !lv.data ? (
        <ErrorState msg={lv.err} onRetry={lv.reload} />
      ) : (
        <>
          {ps.length === 0 ? (
            <EmptyState title="无在持仓位" hint="开仓后此处逐行立账——腿明细点开即看" />
          ) : (
            <table className="rows">
              <thead>
                <tr>
                  <th>合约</th>
                  <th>族</th>
                  <th>方向</th>
                  <th className="num">名义</th>
                  <th>开仓</th>
                  <th className="num">腿</th>
                </tr>
              </thead>
              <tbody>
                {ps.map((p) => (
                  <Fragment key={p.id}>
                    <tr
                      className="rowlink"
                      onClick={() => setOpen((o) => ({ ...o, [Number(p.id)]: !o[Number(p.id)] }))}
                    >
                      <td>{p.instId}</td>
                      <td className="ink-3">{p.template || "—"}</td>
                      <td>{DIRECTION_CN[p.direction] ?? (p.direction || "—")}</td>
                      <td className="num mono">
                        ${p.notionalUsd.toLocaleString("zh-CN", { maximumFractionDigits: 2 })}
                      </td>
                      <td className="mono">{fmtTs(p.openedAt)}</td>
                      <td className="num mono">
                        {p.legs.length === 0 ? "—" : `${open[Number(p.id)] ? "▾" : "▸"}${p.legs.length}`}
                      </td>
                    </tr>
                    {open[Number(p.id)] &&
                      (p.legs.length === 0 ? (
                        <tr key={String(p.id) + ":e"}>
                          <td colSpan={6} className="ink-3">
                            产品仓无订单腿（DCD/结构仓——订阅明细走 venue 产品面）
                          </td>
                        </tr>
                      ) : (
                        <tr key={String(p.id) + ":l"}>
                          <td colSpan={6}>
                            <table className="rows sub">
                              <thead>
                                <tr>
                                  <th>腿向</th>
                                  <th>合约</th>
                                  <th className="num">量</th>
                                  <th className="num">价</th>
                                  <th>态</th>
                                  <th>venue 单号</th>
                                  <th>时</th>
                                </tr>
                              </thead>
                              <tbody>
                                {p.legs.map((l, i) => (
                                  <tr key={i}>
                                    <td>{l.side}</td>
                                    <td>{l.instId}</td>
                                    <td className="num mono">{l.sz}</td>
                                    <td className="num mono">{l.px === 0 ? "—" : l.px}</td>
                                    <td className="ink-3">{l.state}</td>
                                    <td className="mono">{l.venueTicket === 0n ? "—" : String(l.venueTicket)}</td>
                                    <td className="mono">{fmtTs(l.ts)}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </td>
                        </tr>
                      ))}
                  </Fragment>
                ))}
              </tbody>
            </table>
          )}
          {armed.length > 0 && (
            <div className="armed">
              <button
                className="btn tree-head"
                onClick={() => setArmedOpen((v) => !v)}
                aria-expanded={armedOpen}
              >
                <span className={"caret" + (armedOpen ? " open" : "")}>▸</span>
                待命部署 {armed.length} 件（live 待信号——非僵尸）
              </button>
              {armedOpen && (
                <table className="rows sub">
                  <thead>
                    <tr>
                      <th>族</th>
                      <th>合约</th>
                      <th className="num">名义</th>
                      <th>入场</th>
                    </tr>
                  </thead>
                  <tbody>
                    {armed.map((a) => (
                      <tr key={a.id}>
                        <td>{a.template}</td>
                        <td>{a.instId}</td>
                        <td className="num mono">
                          ${a.notionalUsd.toLocaleString("zh-CN", { maximumFractionDigits: 2 })}
                        </td>
                        <td className="mono">{fmtTs(a.enteredAt)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          )}
        </>
      )}
    </Card>
  );
}
