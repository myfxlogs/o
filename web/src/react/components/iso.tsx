// iso.tsx —— 等距投影绘件（/floor 交易厅用，后续 3D 换渲染层同数据绑定位）。
// P=投影/pts=多点串/IsoBox=三面盒/Person=屏幕空间人形立像。
import { useEffect } from "react";

export const T = 30; // 半瓦宽 px
export const P = (x: number, y: number, z = 0): [number, number] => [
  (x - y) * T,
  (x + y) * (T / 2) - z * T,
];
export const pts = (arr: [number, number][]) => arr.map((p) => p.join(",")).join(" ");

// 等距盒：顶面+前 y 面+右 x 面（视角从 +x+y 看入角）
export function IsoBox(props: {
  x: number; y: number; w: number; d: number; h: number; z0?: number;
  top: string; front: string; right: string; dashed?: boolean; op?: number;
}) {
  const { x, y, w, d, h, z0 = 0 } = props;
  const top = pts([P(x, y, z0 + h), P(x + w, y, z0 + h), P(x + w, y + d, z0 + h), P(x, y + d, z0 + h)]);
  const front = pts([P(x, y + d, z0), P(x + w, y + d, z0), P(x + w, y + d, z0 + h), P(x, y + d, z0 + h)]);
  const right = pts([P(x + w, y, z0), P(x + w, y + d, z0), P(x + w, y + d, z0 + h), P(x + w, y, z0 + h)]);
  const dash = props.dashed ? { strokeDasharray: "4 3" } : {};
  const st = { fillOpacity: props.op ?? 1, ...dash };
  return (
    <g>
      <polygon points={top} fill={props.top} stroke="#0b0e14" strokeWidth="0.6" style={st} />
      <polygon points={front} fill={props.front} stroke="#0b0e14" strokeWidth="0.6" style={st} />
      <polygon points={right} fill={props.right} stroke="#0b0e14" strokeWidth="0.6" style={st} />
    </g>
  );
}

// 人形——屏幕空间锚点（底点 x,y,z 投影后向上画），不随 iso 变形的立像。
// 双色：tone=衫色，skin=头色（默认浅肤）——站着的人。
export function Person(props: { x: number; y: number; z?: number; tone: string; breathe?: boolean; skin?: string }) {
  const [px, py] = P(props.x, props.y, props.z ?? 0);
  const c = props.tone, skin = props.skin ?? "#d9a97e";
  return (
    <g className={props.breathe ? "fx-breathe" : ""}>
      <ellipse cx={px} cy={py} rx="10" ry="4" fill="#000" opacity="0.3" />
      <path d={`M ${px - 7} ${py} L ${px - 6} ${py - 20} Q ${px} ${py - 26} ${px + 6} ${py - 20} L ${px + 7} ${py} Z`} fill={c} />
      <circle cx={px} cy={py - 31} r="6.5" fill={skin} stroke="#0b0e14" strokeWidth="0.8" />
    </g>
  );
}

// 坐席人形——躯干式（无腿）：画在桌前椅上，背对镜头面对显示器。
// 锚点=椅面位；desk 之后画则桌沿不遮，桌棱与人交叠出「坐进工位」读感。
export function SeatedPerson(props: { x: number; y: number; tone: string; breathe?: boolean; skin?: string }) {
  const [px, py] = P(props.x, props.y, 0);
  const c = props.tone, skin = props.skin ?? "#d9a97e";
  return (
    <g className={props.breathe ? "fx-breathe" : ""}>
      <ellipse cx={px} cy={py} rx="9" ry="3.5" fill="#000" opacity="0.28" />
      <path d={`M ${px - 8} ${py} L ${px - 7} ${py - 13} Q ${px} ${py - 19} ${px + 7} ${py - 13} L ${px + 8} ${py} Z`} fill={c} />
      <circle cx={px} cy={py - 24} r="6.5" fill={skin} stroke="#0b0e14" strokeWidth="0.8" />
    </g>
  );
}

// 走位人——沿屏幕路径移动的搬运人（animateMotion，dur 后 onDone 卸）。
// 用法：领命/移交等新事实行触发，从白板或门走到目标工位。
export function Walker(props: { path: [number, number][]; dur?: number; tone?: string; onDone: () => void }) {
  const d = props.path.map(([x, y], i) => `${i ? "L" : "M"} ${x} ${y}`).join(" ");
  const dur = props.dur ?? 3400;
  useEffect(() => {
    const t = setTimeout(props.onDone, dur + 60);
    return () => clearTimeout(t);
  });
  return (
    <g>
      <animateMotion dur={`${dur}ms`} path={d} fill="freeze" />
      <g transform="translate(0,-4)">
        <path d="M -6 0 L -5 -17 Q 0 -22 5 -17 L 6 0 Z" fill={props.tone ?? "#5b8def"} />
        <circle cx="0" cy="-27" r="6" fill="#d9a97e" stroke="#0b0e14" strokeWidth="0.8" />
      </g>
    </g>
  );
}
