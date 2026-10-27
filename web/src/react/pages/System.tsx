// System.tsx —— 系统页（T-0170：采集 16 件新鲜度板——服务端判态前端零判据；
// venues 在线树 + 席位心跳 + 进程指标。30s 自动重抓=节拍面活性可见，
// 断流时 Watch 顶栏黄点接管告知）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { useSnap } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { pbDate } from "../components/fmt";

const AGE = (ts: Date) => {
  const s = Math.max(0, (Date.now() - ts.getTime()) / 1000);
  if (s < 60) return `${s.toFixed(0)} 秒前`;
  if (s < 3600) return `${(s / 60).toFixed(0)} 分前`;
  if (s < 86400) return `${(s / 3600).toFixed(1)} 时前`;
  return `${(s / 86400).toFixed(1)} 天前`;
};

export function SystemPage() {
  const h = useSnap(["ev_account", "ev_position"], () =>
    client.getSystemHealth({})
  );
  const ov = useSnap(["ev_position", "ev_account"], () => client.overview({}));
  const [, setTick] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setTick((x) => x + 1), 30000); // 龄文案随拍
    return () => clearInterval(t);
  }, []);

  const hh = (iso: string | undefined) => (iso ? AGE(new Date(iso)) : "—");

  return (
    <div className="page">
      <Card
        title="采集组件"
        extra={
          h.err ? null : (
            <span className="ink-3">
              {(h.data?.components ?? []).filter((c) => c.state === 0).length}/
              {h.data?.components.length ?? 0} 正常
            </span>
          )
        }
      >
        {h.loading && !h.data ? (
          <Skeleton rows={8} />
        ) : h.err && !h.data ? (
          <ErrorState msg={h.err} onRetry={h.reload} />
        ) : (
          <table className="rows">
            <thead>
              <tr>
                <th>组件</th>
                <th>末拍</th>
                <th>判</th>
              </tr>
            </thead>
            <tbody>
              {(h.data?.components ?? []).map((c) => (
                <tr key={c.component}>
                  <td>
                    <StateDot state={c.state} /> {c.component}
                    {c.note ? <div className="ink-3 note">{c.note}</div> : null}
                  </td>
                  <td className="mono">{hh(pbDate(c.lastWrite)?.toISOString())}</td>
                  <td className="ink-3 fact">{c.fact}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>

      <div className="masonry">
      <Card title="进程指标">
        {h.data ? (
          <div className="stat-strip">
            <span>
              帧 <b className="mono">{h.data.frames.toLocaleString("zh-CN")}</b>
            </span>
            <span>
              末帧 <b className="mono">{hh(pbDate(h.data.lastFrame)?.toISOString())}</b>
            </span>
            <span>
              带宽 <b className="mono">{h.data.netKbps.toFixed(0)} kbps</b>
            </span>
            <span>
              拒因{" "}
              {Object.entries(h.data.rejects ?? {}).filter(([, v]) => v > 0).length === 0 ? (
                <b>零</b>
              ) : (
                Object.entries(h.data.rejects ?? {})
                  .filter(([, v]) => v > 0)
                  .map(([k, v]) => (
                    <b key={k} className="u-bad mono">
                      {" "}
                      {k}:{v}
                    </b>
                  ))
              )}
            </span>
          </div>
        ) : (
          <Skeleton rows={1} />
        )}
      </Card>

      <Card title="交易所在线">
        {(ov.data?.venues ?? []).length === 0 ? (
          <Skeleton rows={3} />
        ) : (
          (ov.data?.venues ?? []).map((g) => (
            <div key={g.name} className="venue-group">
              <div className="venue-name">{g.name}</div>
              {(g.channels ?? []).map((ch) => (
                <div key={ch.channel} className="feed-row">
                  <StateDot state={ch.state} />
                  <span>{ch.name}</span>
                  <span className="ink-3 mono">{ch.lastTs ? hh(new Date(ch.lastTs).toISOString()) : "读件缺"}</span>
                </div>
              ))}
            </div>
          ))
        )}
      </Card>

      <Card title="席位在册">
        {(ov.data?.staff ?? []).length === 0 ? (
          <EmptyState title="花名册暂缺" hint="席位注册后此处逐盏亮灯" />
        ) : (
          <div className="chips">
            {(ov.data?.staff ?? []).map((s) => (
              <span key={s.name} className="chip">
                <StateDot state={s.state} /> {s.name}
              </span>
            ))}
          </div>
        )}
      </Card>
      </div>
    </div>
  );
}
