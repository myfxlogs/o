// chart.tsx —— uPlot 薄封装（T-0170：金融时序折线——零依赖 48KB，手机流畅；
// 主题色随令牌，data-theme 切换重绘）。
import { useEffect, useRef } from "react";
import uPlot, { type AlignedData, type Options } from "uplot";
import "uplot/dist/uPlot.min.css";

export function LineChart(props: {
  labels: string[];
  xs: number[]; // unix 秒
  ys: number[];
  height?: number;
  fmtY?: (v: number) => string;
}) {
  const box = useRef<HTMLDivElement>(null);
  const plot = useRef<uPlot | null>(null);

  useEffect(() => {
    if (!box.current || props.xs.length === 0) return;
    const css = getComputedStyle(document.documentElement);
    const ink = css.getPropertyValue("--ink-3").trim() || "#888";
    const line = css.getPropertyValue("--accent").trim() || "#3987e5";
    const fmtY =
      props.fmtY ?? ((v: number) => v.toLocaleString("zh-CN", { maximumFractionDigits: 2 }));
    const opts: Options = {
      width: box.current.clientWidth,
      height: props.height ?? 220,
      cursor: { points: { size: 5 } },
      scales: { x: { time: true } },
      axes: [
        {
          stroke: ink,
          grid: { stroke: css.getPropertyValue("--line").trim() || "#333" },
          ticks: { show: false },
        },
        {
          stroke: ink,
          grid: { stroke: css.getPropertyValue("--line").trim() || "#333" },
          ticks: { show: false },
          values: (_u: uPlot, splits: number[]) => splits.map(fmtY),
        },
      ],
      series: [
        {},
        {
          stroke: line,
          width: 1.6,
          points: { show: props.xs.length < 40 },
          value: (_u: uPlot, v: number) => (v == null ? "—" : fmtY(v)),
        },
      ],
    };
    const data: AlignedData = [Float64Array.from(props.xs), Float64Array.from(props.ys)];
    plot.current = new uPlot(opts, data, box.current);
    const ro = new ResizeObserver(() => {
      if (plot.current && box.current)
        plot.current.setSize({ width: box.current.clientWidth, height: props.height ?? 220 });
    });
    ro.observe(box.current);
    return () => {
      ro.disconnect();
      plot.current?.destroy();
      plot.current = null;
    };
    // 主题切换=令牌变——labels 变化即重建（theme 存 html dataset，重挂由上层 key 驱动亦可）。
  }, [props.xs, props.ys, props.height, props.labels]);

  if (props.xs.length === 0)
    return (
      <div className="empty-state">
        <div className="es-title">曲线还没成形</div>
        <div className="es-hint">净值日拍今天开始积累——每天一个点，周后见形</div>
      </div>
    );
  return <div ref={box} className="chart-box" />;
}
