// floorTokens.tsx —— 控制室模拟盘基件（业主令 11-12：删美工,全语义令牌化）。
// 底板=等距线框网格(只画线);对象=令牌(平行四边形块:态色描边+角点+文行);
// 流动=事实触发的票签(带 #id 沿线滑)。视觉变量只编码态/位/数,不编码「像什么」。
import { useEffect } from "react";
import { P, pts } from "../components/iso";

// 等距线框底板——只画线不填色(保留感知→认知→记录的空间编码)。
export function IsoGrid(props: { w: number; d: number }) {
  const { w, d } = props;
  return (
    <g>
      <polygon points={pts([P(0, 0), P(w, 0), P(w, d), P(0, d)])} fill="#141925" stroke="#232b3c" strokeWidth="1.2" />
      {Array.from({ length: w + 1 }, (_, i) => (
        <line key={"x" + i} x1={P(i, 0)[0]} y1={P(i, 0)[1]} x2={P(i, d)[0]} y2={P(i, d)[1]} stroke="#1c2331" strokeWidth="0.5" />
      ))}
      {Array.from({ length: d + 1 }, (_, i) => (
        <line key={"y" + i} x1={P(0, i)[0]} y1={P(0, i)[1]} x2={P(w, i)[0]} y2={P(w, i)[1]} stroke="#1c2331" strokeWidth="0.5" />
      ))}
    </g>
  );
}

// 区块题注——小区块名(感知哨兵/器官工位…),dim 小字。
export function ZoneCap(props: { x: number; y: number; label: string }) {
  const [px, py] = P(props.x, props.y);
  return <text x={px} y={py} className="iso-cap dim" fontSize="8.5">{props.label}</text>;
}

// 令牌——平行四边形块:tone=描边态色,lines=块内文行(第一行=名),
// sub=前沿下小字(事实锚等),dim=毛坯淡影。文行总高按行距收进块内。
export function Cell(props: {
  x: number; y: number; w: number; d: number; tone: string;
  lines?: string[]; sub?: string; dim?: boolean; fs?: number;
}) {
  const { x, y, w, d, tone, sub } = props;
  const lines = props.lines ?? [];
  const [cx, cy] = P(x + w / 2, y + d / 2);
  const [sx, sy] = P(x + w / 2, y + d);
  const [dx, dy] = P(x + 0.16, y + d - 0.08);
  const n = lines.length, fs = props.fs ?? 9.5, lh = fs + 3;
  return (
    <g opacity={props.dim ? 0.42 : 1}>
      <polygon points={pts([P(x, y), P(x + w, y), P(x + w, y + d), P(x, y + d)])}
        fill="#181e2b" stroke={tone} strokeWidth="1.1" />
      <circle cx={dx} cy={dy} r="2.2" fill={tone} />
      {lines.map((t, i) => (
        <text key={i} x={cx} y={cy + (i - (n - 1) / 2) * lh + fs * 0.36}
          className={i === 0 ? "iso-cap" : "iso-cap dim"} fontSize={i === 0 ? fs : fs - 1}
          textAnchor="middle">{t}</text>
      ))}
      {sub && <text x={sx} y={sy + 12} className="iso-cap dim" fontSize="7.5" textAnchor="middle">{sub}</text>}
    </g>
  );
}

// 票签——事实触发的搬运令牌:带 #id 的小块沿屏幕路径滑(蓝=取钉,绿=送档)。
export function Packet(props: { path: [number, number][]; dur?: number; tone: string; id: number; onDone: () => void }) {
  const d = props.path.map(([x, y], i) => `${i ? "L" : "M"} ${x} ${y}`).join(" ");
  const dur = props.dur ?? 3400;
  useEffect(() => {
    const t = setTimeout(props.onDone, dur + 80);
    return () => clearTimeout(t);
  }, [dur]);
  return (
    <g>
      <animateMotion dur={`${dur}ms`} path={d} fill="freeze" />
      <g transform="translate(0,-9)">
        <rect x="-17" y="-7" width="34" height="14" rx="3" fill="#10151f" stroke={props.tone} strokeWidth="1.4" />
        <text x="0" y="3" textAnchor="middle" fontSize="8" fill={props.tone}>#{props.id}</text>
      </g>
    </g>
  );
}
