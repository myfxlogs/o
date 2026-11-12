// octopus.tsx —— 器官章鱼（业主令：floor 画八爪鱼,每器官一腕连脑）。
// 语义绑定（无装饰像素）:
//   脑 = 唤知会话席——在班睁眼泛光、下班闭眼,点脑=会话席聚焦
//   腕 = GHOSTS 器官×N(动态,不硬写 8)——色/形=ghostSt 四态:
//     done=实腕(绿,带吸盘) build=再生中(金短腕,动虚线)
//     pin=立项(蓝虚线骨架) ghost=毛坯(灰腕芽)
//   光流 = 该器官域 ops_event 窗内命中数——越活搏动越急,流向腕尖→脑(感知上行)
//   点腕 = 同瓦片聚焦(gh<i>)
import type { Tip } from "../pages/floorFocus";

export type Arm = {
  i: number;
  name: string;
  st: "ghost" | "pin" | "build" | "done";
  hits: number; // ops_event 窗内该器官域命中数
  flash?: boolean; // 新行到达脉冲(0.9s 抽腕)
};
export const ST_TXT: Record<string, string> = { ghost: "毛坯", pin: "已立项", build: "在建", done: "已完工" };

const CX = 235, CY = 92, RX = 54, RY = 36;
const RAD = Math.PI / 180;
type P = { x: number; y: number };
// 二次贝塞尔取点(吸盘/标签落位同一路径)
const qb = (p0: P, c: P, p1: P, t: number): P => {
  const u = 1 - t;
  return { x: u * u * p0.x + 2 * u * t * c.x + t * t * p1.x, y: u * u * p0.y + 2 * u * t * c.y + t * t * p1.y };
};

export function Octopus(props: { arms: Arm[]; awake: boolean; brainFlash?: boolean; brainNote?: string; brainAct?: string; onPick: (t: Tip) => void }) {
  const n = props.arms.length;
  const fmt = (p: P) => `${p.x.toFixed(1)} ${p.y.toFixed(1)}`;
  return (
    <svg className="octo" viewBox="0 0 470 308" role="img" aria-label="器官章鱼">
      {/* 腕先画——起点藏进脑缘下,天然连脑 */}
      {props.arms.map((a) => {
        const th = (20 + a.i * 140 / Math.max(1, n - 1)) * RAD; // 扇面 20°→160°,90°正下
        const len = a.st === "ghost" ? 70 : a.st === "build" ? 108 : 148; // 在建=再生中短腕,毛坯=腕芽
        const curl = (a.i % 2 ? 1 : -1) * len * 0.16; // 交替内勾——章鱼腕的自然卷
        const p0 = { x: CX + RX * Math.cos(th) * 0.92, y: CY + RY * Math.sin(th) * 0.92 };
        const perp = { x: -Math.sin(th) * curl, y: Math.cos(th) * curl };
        const c = { x: p0.x + Math.cos(th) * len * 0.52 + perp.x, y: p0.y + Math.sin(th) * len * 0.52 + perp.y };
        const p1 = { x: p0.x + Math.cos(th) * len + perp.x * 0.35, y: p0.y + Math.sin(th) * len + perp.y * 0.35 };
        const arm = `M ${fmt(p0)} Q ${fmt(c)} ${fmt(p1)}`;
        const flow = `M ${fmt(p1)} Q ${fmt(c)} ${fmt(p0)}`; // 反向——光流腕尖→脑
        const tip = { x: p1.x + Math.cos(th) * 6, y: p1.y + Math.sin(th) * 10 + 8 };
        const suckers = a.st === "done" ? [0.42, 0.58, 0.74, 0.88].map((t) => qb(p0, c, p1, t)) : [];
        const dur = a.hits > 0 ? Math.max(1.1, 7 - a.hits * 1.1).toFixed(1) : null;
        return (
          <g key={a.name} className={`octo-arm st-${a.st}${a.flash ? " flash" : ""}`}
            style={{ transformOrigin: `${p0.x}px ${p0.y}px` }}
            onClick={() => props.onPick({ k: "gh" + a.i, x: 0, y: 0, title: `工位 · ${a.name}`, body: [], to: "/decision" })}>
            <title>{a.name} · {ST_TXT[a.st]}{a.hits ? ` · 窗内信号×${a.hits}` : ""}</title>
            <path d={arm} className="hit" />
            <path d={arm} className="arm" />
            {dur && <path d={flow} className="flow" style={{ animationDuration: `${dur}s` }} />}
            {suckers.map((p, j) => <circle key={j} cx={p.x} cy={p.y} r={1.7} className="octo-sucker" />)}
            <text x={tip.x} y={tip.y} textAnchor="middle" className="octo-name">{a.name}</text>
            <text x={tip.x} y={tip.y + 10} textAnchor="middle" className="octo-st">{ST_TXT[a.st]}</text>
          </g>
        );
      })}
      {/* 脑=会话席——认领件贴脑下,新行到脑闪 */}
      <g className={`octo-brain${props.awake ? " awake" : ""}${props.brainFlash ? " flash" : ""}`}
        onClick={() => props.onPick({ k: "desk", x: 0, y: 0, title: "会话席", body: [], to: "/device" })}>
        <title>脑 · 唤知会话席——{props.awake ? "在班" : "下班"}</title>
        {props.awake && <ellipse cx={CX} cy={CY} rx={RX + 7} ry={RY + 6} className="octo-halo" />}
        <path className="mantle"
          d={`M ${CX - RX} ${CY + 6} C ${CX - RX - 8} ${CY - 30}, ${CX - 30} ${CY - RY - 16}, ${CX} ${CY - RY - 16} C ${CX + 30} ${CY - RY - 16}, ${CX + RX + 8} ${CY - 30}, ${CX + RX} ${CY + 6} C ${CX + RX + 4} ${CY + 26}, ${CX + 30} ${CY + RY}, ${CX} ${CY + RY} C ${CX - 30} ${CY + RY}, ${CX - RX - 4} ${CY + 26}, ${CX - RX} ${CY + 6} Z`} />
        {props.awake ? (<>
          <circle cx={CX - 17} cy={CY - 4} r="4.6" className="octo-eye" />
          <circle cx={CX + 17} cy={CY - 4} r="4.6" className="octo-eye" />
          <circle cx={CX - 15.5} cy={CY - 5.5} r="1.4" className="octo-pupil" />
          <circle cx={CX + 18.5} cy={CY - 5.5} r="1.4" className="octo-pupil" />
        </>) : (<>
          <path d={`M ${CX - 23} ${CY - 3} q 6 5 12 0`} className="octo-eye-shut" />
          <path d={`M ${CX + 11} ${CY - 3} q 6 5 12 0`} className="octo-eye-shut" />
        </>)}
        <text x={CX} y={CY + 20} textAnchor="middle" className="octo-brain-st">{props.awake ? "脑·在班" : "脑·下班"}</text>
        {props.brainNote && <text x={CX} y={CY - RY - 24} textAnchor="middle" className="octo-brain-note">{props.brainNote}</text>}
        {props.brainAct && <text x={CX} y={CY - RY - 40} textAnchor="middle" className="octo-brain-act">{props.brainAct}</text>}
      </g>
    </svg>
  );
}
