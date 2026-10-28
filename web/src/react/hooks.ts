// hooks.ts —— 信号重取钩（T-0220b：useSnap 退役——推帧面（统一流水默认视图/
// 历史订单首页/主机资源）走 store 分片订阅（store.ts useSlice，一跳零重拉）；
// 未载荷化聚合面（overview/health/reports/equity/cashflows）走本钩=抓一次+
// topics 命中帧重取+key（请求参数指纹——参数变=重取）；诚实空态律=错误显形
// 不假零）。
import { useCallback, useEffect, useRef, useState } from "react";
import { onData } from "./stream";

export type Live<T> = {
  data: T | null;
  err: string | null;
  loading: boolean;
  reload: () => void;
};

export function useLive<T>(
  topics: string[],
  fetcher: () => Promise<T>,
  key?: unknown
): Live<T> {
  const [data, setData] = useState<T | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    fetcherRef
      .current()
      .then((d) => {
        if (!alive) return;
        setData(d);
        setErr(null);
      })
      .catch((e: unknown) => {
        if (!alive) return;
        setErr(e instanceof Error ? e.message : String(e));
      })
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [tick, key]);

  useEffect(() => {
    if (topics.length === 0) return;
    return onData((v) => {
      const topic = (v as { topic?: string } | undefined)?.topic;
      if (topic && topics.includes(topic)) setTick((t) => t + 1);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, topics);

  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, err, loading, reload };
}
