// System.tsx —— 系统页（T-0173：交易所在线树（SG 视角——镜像+SG 本地面）
// +席位在册。采集组件健康已独立「采集」页（两台主机两大瓦片，各管各的）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { useSnap } from "../hooks";
import { Card, EmptyState, Skeleton, StateDot } from "../components/ui";

const AGE = (ts: Date) => {
  const s = Math.max(0, (Date.now() - ts.getTime()) / 1000);
  if (s < 60) return `${s.toFixed(0)} 秒前`;
  if (s < 3600) return `${(s / 60).toFixed(0)} 分前`;
  if (s < 86400) return `${(s / 3600).toFixed(1)} 时前`;
  return `${(s / 86400).toFixed(1)} 天前`;
};

export function SystemPage() {
  const ov = useSnap(["ev_position", "ev_account"], () => client.overview({}));
  const [, setTick] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setTick((x) => x + 1), 30000);
    return () => clearInterval(t);
  }, []);

  return (
    <div className="page">
      <Card title="交易所在线（SG 视角）">
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
                  <span className="ink-3 mono">
                    {ch.lastTs ? AGE(new Date(ch.lastTs)) : "读件缺"}
                  </span>
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
                <StateDot state={s.state} breathe={s.state !== 2} /> {s.name}
              </span>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
