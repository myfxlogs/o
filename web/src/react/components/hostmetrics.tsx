// hostmetrics.tsx —— 主机资源卡（T-0173 业主令：CPU/内存/磁盘/网络——
// 各机自报 30s 拍，GetHostMetrics 契约；stale=拍断显陈不假值；
// 用量条 ≥80% 转警示色。T-0220b：ev_hostmetrics 推帧分片订阅——一跳渲染）。
import { useEffect } from "react";
import { client } from "../api";
import { hydrate, useSlice } from "../store";
import { Card, ErrorState, Skeleton, StateDot } from "./ui";
import { AGE } from "./panel";
import { pbDate } from "./fmt";

function Bar(props: { label: string; pct: number; text: string }) {
  const warn = props.pct >= 80;
  return (
    <div className="hm-bar">
      <div className="hm-row">
        <span className="ink-3">{props.label}</span>
        <span className="mono">{props.text}</span>
      </div>
      <div className="hm-track">
        <div
          className={"hm-fill" + (warn ? " u-bad" : "")}
          style={{ width: Math.min(100, props.pct) + "%" }}
        />
      </div>
    </div>
  );
}

function HostTile(props: {
  hm: NonNullable<
    Awaited<ReturnType<typeof client.getHostMetrics>>
  >["hosts"][number];
}) {
  const hm = props.hm;
  const d = pbDate(hm.ts);
  const memPct = hm.memTotalMb > 0 ? (hm.memUsedMb / hm.memTotalMb) * 100 : 0;
  const dskPct = hm.diskTotalGb > 0 ? (hm.diskUsedGb / hm.diskTotalGb) * 100 : 0;
  return (
    <div className="hm-tile">
      <div className="hm-head">
        <StateDot state={hm.stale ? 2 : 0} breathe={!hm.stale} />
        <b>{hm.host === "dev" ? "OCI" : hm.host.toUpperCase()}</b>
        <span className="ink-3" style={{ fontSize: 12 }}>
          {hm.stale ? "拍断（>2min）" : d ? AGE(d) : ""}
          {hm.cores ? ` · ${hm.cores} 核` : ""}
        </span>
      </div>
      {hm.stale ? (
        <div className="ink-3" style={{ fontSize: 12, padding: "6px 0" }}>
          自报停拍——查该机采集进程/ROSTER 隧道
        </div>
      ) : (
        <>
          <Bar
            label="CPU"
            pct={hm.cpuPct}
            text={`${hm.cpuPct.toFixed(0)}%`}
          />
          <Bar
            label="内存"
            pct={memPct}
            text={`${(hm.memUsedMb / 1024).toFixed(1)} / ${(hm.memTotalMb / 1024).toFixed(1)} GB`}
          />
          <Bar
            label="磁盘"
            pct={dskPct}
            text={`${hm.diskUsedGb.toFixed(0)} / ${hm.diskTotalGb.toFixed(0)} GB`}
          />
          <div className="hm-row" style={{ marginTop: 6 }}>
            <span className="ink-3">网络</span>
            <span className="mono">
              ↓{hm.netRxKbps.toFixed(0)} ↑{hm.netTxKbps.toFixed(0)} kbps
            </span>
          </div>
          <div className="hm-row">
            <span className="ink-3">运行</span>
            <span className="mono">{hm.uptimeHours.toFixed(0)} h</span>
          </div>
        </>
      )}
    </div>
  );
}

export function HostMetricsCard() {
  const h = useSlice("hostMetrics");
  const retry = () =>
    hydrate("hostMetrics", () => client.getHostMetrics({}));
  useEffect(() => {
    retry();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return (
    <Card title="主机资源">
      {h.loading && !h.data ? (
        <Skeleton rows={4} />
      ) : h.err && !h.data ? (
        <ErrorState msg={h.err} onRetry={retry} />
      ) : (h.data?.hosts ?? []).length === 0 ? (
        <div className="ink-3" style={{ fontSize: 13, padding: "6px 0" }}>
          首拍未落——各机自报 30s 拍，重启后一分钟内成形
        </div>
      ) : (
        <div className="host-grid">
          {(h.data?.hosts ?? []).map((hm) => (
            <HostTile key={hm.host} hm={hm} />
          ))}
        </div>
      )}
    </Card>
  );
}
