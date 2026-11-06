// brokerboard.tsx —— T-0407 座舱 broker 接入卡（票面：venueboard 独立卡
// 同页跟形——在册 broker 列表+「添加 broker」表单）：
//   · 列表=ListBrokerOnboard 四源并集行（venue/协议/login/接入态/在跑策略
//     数/末拍注记+败径标——判态服务端派生，前端零判据纯渲染）；
//   · 表单字段=venue 名/协议类型（MT5/MT4 下拉）/登录号/密码/服务器/备注
//     ——凭据明文只进请求体（type=password 不回显不持久，提交即弃；
//     localStorage/console/URL 零触碰——domtest 钉）；
//   · 提交=SubmitBrokerCred 窄写 RPC——app_secrets <venue>.<proto>.* 族落
//     即 T-0379 四拍链自动接管（探测→草案→map 闸→paper 激活），本卡只呈
//     各拍状态/败因（ops_event「venue接入」行只读消费）；
//   · MT4 族=凭据在册链未接如实显形（链 MT5 族限——不假冒接入态）。
// 刷新=ev_pipeline（ops_event 插行帧——每拍落行即重取）+ev_strategy
// （在跑策略数源）+提交后 reload 同帧直反。
import { useState, type FormEvent } from "react";
import { client, errText } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";
import type { BrokerOnboardRow } from "../../gen/dashboardv1/core_pb";

// 人话字典（纯显示映射非判据——接入态键→中文直译；未知值原样透出不吞）。
const STATE_CN: Record<string, string> = {
	registered: "已注册·候探测",
	probed: "已探测",
	mapped: "映射就绪",
	paper: "paper 激活",
	candidate: "候选",
	measuring: "度量中",
	recommended: "建议注册",
	rejected: "已拒",
	kyc_blocked: "KYC受限",
	adopted: "已采纳",
	dismissed: "已驳回",
	parked: "已停放",
};
const stateTxt = (r: BrokerOnboardRow): string => {
	if (r.proto === "mt4" && !r.state) return "凭据在册·MT4 链未接";
	if (!r.state) return "凭据在册·候发现";
	return STATE_CN[r.state] ?? r.state;
};

export type BrokerForm = {
	venue: string;
	proto: string;
	login: string;
	password: string;
	server: string;
	note: string;
};

// BrokerOnboardView 纯渲件（DOM 桩测面——容器喂行+回调；表单开合=纯 UI 态）。
export function BrokerOnboardView(props: {
	rows: BrokerOnboardRow[];
	err?: string | null;
	busy?: boolean;
	onSubmit?: (f: BrokerForm) => void;
}) {
	const submit = (e: FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		if (!props.onSubmit) return;
		const f = new FormData(e.currentTarget);
		props.onSubmit({
			venue: String(f.get("venue") ?? ""),
			proto: String(f.get("proto") ?? "mt5"),
			login: String(f.get("login") ?? ""),
			password: String(f.get("password") ?? ""),
			server: String(f.get("server") ?? ""),
			note: String(f.get("note") ?? ""),
		});
		// 凭据字段提交即弃——reset 清表单值，DOM/状态零驻留。
		e.currentTarget.reset();
	};
	return (
		<Card title="broker 接入">
			{props.rows.length === 0 ? (
				<EmptyState title="在册 broker 空"
					hint="下方表单提交凭据——MT5 族凭据落库即接入链自动接管" />
			) : (
				<table className="rows">
					<thead>
						<tr><th>broker</th><th>协议</th><th>login</th><th>接入态</th><th>在跑策略</th><th>末拍注记</th></tr>
					</thead>
					<tbody>
						{props.rows.map((r) => (
							<tr key={r.venue}>
								<td>{r.venue}</td>
								<td className="ink-3">{(r.proto || "—").toUpperCase()}</td>
								<td className="mono">{r.login || "—"}</td>
								<td>
									{r.beatFailed
										? <span className="chip chip-critical">接入败</span>
										: stateTxt(r)}
								</td>
								<td className="num mono">{String(r.strategies)}</td>
								<td className="ink-3" title={r.lastBeatAt
									? new Date(Number(r.lastBeatAt.seconds) * 1000).toLocaleString("zh-CN", { hour12: false })
									: undefined}>
									{r.beatNote || "—"}
								</td>
							</tr>
						))}
					</tbody>
				</table>
			)}
			{props.onSubmit && (
				<details style={{ marginTop: 10 }}>
					<summary className="dim" style={{ cursor: "pointer" }}>添加 broker</summary>
				<form onSubmit={submit} style={{ display: "flex", gap: 6, flexWrap: "wrap", marginTop: 10 }}>
					<input name="venue" className="modal-input" required autoComplete="off"
						placeholder="broker 名（小写数线横）" style={{ width: 150 }}
						pattern="[a-z0-9_\-]+" title="venue token——凭据键前缀（小写字母/数字/-/_）" />
					<select name="proto" className="modal-input" defaultValue="mt5" title="协议类型">
						<option value="mt5">MT5</option>
						<option value="mt4">MT4</option>
					</select>
					<input name="login" className="modal-input" required autoComplete="off"
						inputMode="numeric" pattern="[0-9]+" placeholder="登录号"
						style={{ width: 120 }} title="MT 户号——纯数字" />
					<input name="password" className="modal-input" required type="password"
						autoComplete="new-password" placeholder="密码" style={{ width: 140 }}
						title="凭据明文只进请求体——提交即弃不回显不落持久面" />
					<input name="server" className="modal-input" required autoComplete="off"
						placeholder="服务器" style={{ width: 170 }} title="broker 服务器名" />
					<input name="note" className="modal-input" autoComplete="off"
						placeholder="备注（可空）" style={{ width: 140 }} />
					<button type="submit" className="btn-primary" disabled={props.busy}>提交凭据</button>
				</form>
				</details>
			)}
			{props.err && <p className="dim">操作未成：{props.err}</p>}
		</Card>
	);
}

// BrokerOnboardCard 容器件（取数+写径装配——ev_pipeline=每拍落行帧重取；
// 写径裸调不入合并窗，提交后 reload 同帧直反真值）。
export function BrokerOnboardCard() {
	const live = useLive(["ev_pipeline", "ev_strategy"], () => client.listBrokerOnboard({}));
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState<string | null>(null);
	const run = (fn: () => Promise<unknown>) => {
		setBusy(true);
		setErr(null);
		return fn()
			.then(() => live.reload())
			.catch((e: unknown) => setErr(errText(e)))
			.finally(() => setBusy(false));
	};
	if (live.loading && !live.data) {
		return <Card title="broker 接入"><Skeleton rows={2} /></Card>;
	}
	if (!live.data) {
		return (
			<Card title="broker 接入">
				<ErrorState msg={live.err ?? "读败"} onRetry={live.reload} />
			</Card>
		);
	}
	return (
		<BrokerOnboardView
			rows={live.data.rows}
			err={err}
			busy={busy}
			onSubmit={(f) => void run(() => client.submitBrokerCred(f))}
		/>
	);
}
