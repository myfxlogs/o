// ErrorBoundary —— 卡级错误边界（T-0173：单卡运行时故障不再连坐卸载整页
// ——「卡不见了」病灶防线；边界内显读败+重试，页内其余卡不受累）。
import { Component, type ReactNode } from "react";

type Props = { label: string; children: ReactNode };
type State = { err: Error | null };

export class ErrorBoundary extends Component<Props, State> {
  state: State = { err: null };
  static getDerivedStateFromError(err: Error): State {
    return { err };
  }
  componentDidUpdate(prev: Props) {
    // children 换卡（路由/重排）时清错重挂
    if (prev.children !== this.props.children && this.state.err) {
      this.setState({ err: null });
    }
  }
  render() {
    if (this.state.err) {
      return (
        <section className="card">
          <h3>{this.props.label}</h3>
          <div className="err-state">
            <span>渲染故障：{String(this.state.err.message || this.state.err)}</span>
            <button className="btn" onClick={() => this.setState({ err: null })}>
              重试
            </button>
          </div>
        </section>
      );
    }
    return this.props.children;
  }
}
