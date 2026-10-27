// ui.tsx —— 卡面原语（T-0170：Card/骨架/错误态/状态点——诚实空态律：
// 错误显文案+重试钮，禁假零；骨架=卡面独立不整页灰）。
import type { ReactNode } from "react";

export function Card(props: {
  title: ReactNode;
  extra?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={"card " + (props.className ?? "")}>
      <div className="card-h">
        <h2>{props.title}</h2>
        {props.extra}
      </div>
      {props.children}
    </section>
  );
}

export function Skeleton(props: { rows?: number }) {
  return (
    <div className="skel" aria-hidden>
      {Array.from({ length: props.rows ?? 3 }, (_, i) => (
        <div key={i} className="skel-row" />
      ))}
    </div>
  );
}

export function ErrorState(props: { msg: string; onRetry: () => void }) {
  return (
    <div className="err-state" role="alert">
      <span>读件缺——{props.msg}</span>
      <button className="btn" onClick={props.onRetry}>
        重试
      </button>
    </div>
  );
}

// StateDot 三态点（0绿/1黄/2红——语义在旁注文字，色点不带字面）。
export function StateDot(props: { state: number; title?: string }) {
  const cls = props.state === 0 ? "s0" : props.state === 1 ? "s1" : "s2";
  return <span className={"dot " + cls} title={props.title ?? ""} />;
}

export function EmptyRow(props: { text: string }) {
  return <div className="empty-row">{props.text}</div>;
}
