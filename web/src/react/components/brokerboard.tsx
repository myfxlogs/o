// brokerboard.tsx —— T-0407/T-0430/T-0432 座舱 broker 接入卡（票面：三步向导
// verify-then-write——mt-accounts skill 钦定形：①venue ②凭据 ③确认验连；
// T-0432 扩步④符号映射——映射=确认件非推断件）：
//   · 列表=ListBrokerOnboard 并集行（venue/协议/login/接入态 phase/在跑
//     策略数/末拍·败因——phase 判态服务端派生 registered→connecting→
//     verified/failed，前端零判据纯渲染）；
//   · 向导步1=类型（MT5/CEX——MT4 不处理业主令 2026-11-07 选项不存）+
//     venue（MT5 手填+「搜索面待补」注记；CEX=okx 首档枚举）；步2=凭据
//     （MT5：login/password/server+host 选填双轨；CEX：api_key/api_secret/
//     passphrase）；步3=预览（凭据掩码不回显）+提交——后端验连成功才落库
//     （败=即回原因零落表，本卡显败因留步返改）；步④=符号映射（MT5 专有
//     ——官名清单→候选机造下拉（score 排序服务端出）→席位挑→硬验落行
//     （拒收即回零落行），CEX 无此步；
//   · 凭据明文只进请求体（type=password 不回显不持久，提交成功即弃；
//     localStorage/console/URL 零触碰——domtest 钉）；
//   · 提交=SubmitBrokerCred 窄写 RPC——verify-then-write 过闸凭据落
//     app_secrets <venue>.<proto>.* 族即 T-0379 四拍链自动接管，本卡只呈
//     phase/败因（ops_event 行只读消费）；
//   · MT4 族=存量遗留读面兼容如实显形（「MT4 链未接」——不假冒接入态）。
// 刷新=ev_pipeline（ops_event 插行帧——每拍落行即重取）+ev_strategy
// （在跑策略数源）+提交成功后 reload 同帧直反。
import { useState, type FormEvent, type ChangeEvent } from "react";
import { client, errText } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";
import { BrokerOnboardList } from "./brokeronboardlist";
import { SymbolMappingStep, type MappingQuery, type MappingSubmit } from "./brokermap";
import type { SymbolCandidate } from "../../gen/dashboardv1/core_pb";
import type { BrokerOnboardRow } from "../../gen/dashboardv1/core_pb";

// 掩码显示（凭据预览钉——长度可见值不可见，raw 值零渲染）。
const mask = (v?: string) => (v ? "••••••（" + v.length + " 位）" : "—");

export type BrokerForm = {
	venue: string;
	proto: string; // mt5|cex（mt4 不处理）
	login: string;
	password: string;
	server: string;
	host: string;
	note: string;
	apiKey: string;
	apiSecret: string;
	passphrase: string;
	canonical: string; // 步④符号映射面（官名——席位挑定件）
	venueSymbol: string; // 步④席位挑定（候选下拉——非机选）
};

const STEP_CN = ["①venue", "②凭据", "③确认", "④符号映射"];
const CEX_VENUES = ["okx"]; // 首档枚举（他所见 venue 册再扩——后端册闸同源）

// BrokerOnboardView 纯渲件（DOM 桩测面——容器喂行+回调；向导步进=纯 UI 态，
// initial* props=domtest 注面——生产调用零携带）。
export function BrokerOnboardView(props: {
	rows: BrokerOnboardRow[];
	onSubmit?: (f: BrokerForm) => Promise<void> | void;
	initialStep?: number;
	initialDraft?: Partial<BrokerForm>;
	initialErr?: string | null;
	// T-0432 步④符号映射回调（容器装配 RPC——View 纯选转零直连）。
	loadUniverse?: () => Promise<string[]>;
	onMappingQuery?: (f: MappingQuery) => Promise<{ rows: SymbolCandidate[]; scanned: number }>;
	onSubmitMapping?: (f: MappingSubmit) => Promise<void>;
}) {
	const [step, setStep] = useState(props.initialStep ?? 1);
	const [draft, setDraft] = useState<Partial<BrokerForm>>(props.initialDraft ?? {});
	const [subErr, setSubErr] = useState<string | null>(props.initialErr ?? null);
	const [busy, setBusy] = useState(false);
	const set =
		(k: keyof BrokerForm) =>
		(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
			setDraft((d) => ({ ...d, [k]: e.target.value }));
	const kind = draft.proto === "cex" ? "cex" : "mt5";
	// 步进闸（轻形验——硬闸在后端 verify/形闸，前端只拦空步）。
	const stepFull =
		step === 1
			? Boolean(draft.venue)
			: step === 2
				? kind === "mt5"
					? Boolean(draft.login && draft.password && draft.server)
					: Boolean(draft.apiKey && draft.apiSecret && draft.passphrase)
				: true;
	const maxStep = kind === "mt5" ? 4 : 3;
	const submit = async (e: FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		if (!props.onSubmit || busy) return;
		setBusy(true);
		setSubErr(null);
		try {
			await props.onSubmit({
				venue: draft.venue ?? "",
				proto: kind,
				login: draft.login ?? "",
				password: draft.password ?? "",
				server: draft.server ?? "",
				host: draft.host ?? "",
				note: draft.note ?? "",
				apiKey: draft.apiKey ?? "",
				apiSecret: draft.apiSecret ?? "",
				passphrase: draft.passphrase ?? "",
				canonical: "",
				venueSymbol: "",
			});
			// 凭据字段提交成功即弃（DOM/状态零驻留——domtest 钉）；MT5 进位步④
			// 符号映射（venue/login 件保留——映射 RPC 必携），CEX 回步1。
			setDraft(kind === "mt5" ? (d) => ({ venue: d.venue, proto: d.proto, login: d.login }) : {});
			setStep(kind === "mt5" ? 4 : 1);
		} catch (er) {
			setSubErr(errText(er)); // 败因即回显形——留步3可返步改（凭据未落库）
		} finally {
			setBusy(false);
		}
	};
	const next = () => stepFull && setStep(Math.min(maxStep, step + 1));
	const back = () => setStep(Math.max(1, step - 1));
	return (
		<Card title="broker 接入">
			{props.rows.length === 0 ? (
				<EmptyState title="在册 broker 空"
					hint="下方向导提交凭据——验连成功才落库，MT5 族落库即接入链自动接管" />
			) : (
				<BrokerOnboardList rows={props.rows} />
			)}
			{props.onSubmit && (
					<details open style={{ marginTop: 10 }}>
						<summary className="dim" style={{ cursor: "pointer" }}>添加 broker（三步向导——验连成功才落库；MT5 续步④符号映射）</summary>
					<form onSubmit={submit} style={{ marginTop: 10 }}>
						<div style={{ marginBottom: 8 }}>
							{STEP_CN.map((s, i) => (
								<span key={s} className={"chip " + (step === i + 1 ? "chip-warn" : "")}
									title={"步" + (i + 1)}>{s}</span>
							))}
						</div>
						{step === 1 && (
							<div style={{ display: "flex", gap: 6, flexWrap: "wrap", alignItems: "center" }}>
								<select className="modal-input" value={kind} onChange={set("proto")} title="接入类型">
									<option value="mt5">MT5</option>
									<option value="cex">CEX 交易所</option>
								</select>
								{kind === "mt5" ? (
									<input className="modal-input" value={draft.venue ?? ""} onChange={set("venue")}
										autoComplete="off" placeholder="broker 名（小写数线横）" style={{ width: 170 }}
										pattern="[a-z0-9_\-]+" title="venue token——凭据键前缀（小写字母/数字/-/_）" />
								) : (
									<select className="modal-input" value={draft.venue ?? ""} onChange={set("venue")}
										title="交易所（首档枚举）">
										<option value="">—选交易所—</option>
										{CEX_VENUES.map((v) => <option key={v} value={v}>{v.toUpperCase()}</option>)}
									</select>
								)}
								<span className="ink-3">
									{kind === "mt5" ? "broker 搜索面待补——venue token 手填" : "首档 okx——他所候册再扩"}
								</span>
							</div>
						)}
						{step === 2 && kind === "mt5" && (
							<div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
								<input className="modal-input" value={draft.login ?? ""} onChange={set("login")}
									required autoComplete="off" inputMode="numeric" pattern="[0-9]+" placeholder="登录号"
									style={{ width: 120 }} title="MT 户号——纯数字" />
								<input className="modal-input" value={draft.password ?? ""} onChange={set("password")}
									required type="password" autoComplete="new-password" placeholder="密码"
									style={{ width: 140 }} title="凭据明文只进请求体——提交即弃不回显不落持久面" />
								<input className="modal-input" value={draft.server ?? ""} onChange={set("server")}
									required autoComplete="off" placeholder="服务器名" style={{ width: 170 }}
									title="broker 服务器名（ConnectEx 名径）" />
								<input className="modal-input" value={draft.host ?? ""} onChange={set("host")}
									autoComplete="off" placeholder="host:port（选填）" style={{ width: 190 }}
									title="host:port 多候选逗号分隔——Connect 主径备轨（选填）" />
							</div>
						)}
						{step === 2 && kind === "cex" && (
							<div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
								<input className="modal-input" value={draft.apiKey ?? ""} onChange={set("apiKey")}
									required autoComplete="off" placeholder="API Key" style={{ width: 200 }}
									title="凭据明文只进请求体——提交即弃不回显不落持久面" />
								<input className="modal-input" value={draft.apiSecret ?? ""} onChange={set("apiSecret")}
									required type="password" autoComplete="new-password" placeholder="API Secret"
									style={{ width: 200 }} title="凭据明文只进请求体——提交即弃不回显不落持久面" />
								<input className="modal-input" value={draft.passphrase ?? ""} onChange={set("passphrase")}
									required type="password" autoComplete="new-password" placeholder="Passphrase"
									style={{ width: 160 }} title="凭据明文只进请求体——提交即弃不回显不落持久面" />
							</div>
						)}
						{step === 3 && (
							<table className="rows" style={{ maxWidth: 480 }}>
								<tbody>
									<tr><td>类型</td><td>{kind === "mt5" ? "MT5" : "CEX 交易所"}</td></tr>
									<tr><td>venue</td><td className="mono">{draft.venue}</td></tr>
									{kind === "mt5" ? (
										<>
											<tr><td>login</td><td className="mono">{draft.login}</td></tr>
											<tr><td>服务器</td><td className="mono">{draft.server}</td></tr>
											<tr><td>host</td><td className="mono">{draft.host || "—"}</td></tr>
											<tr><td>密码</td><td>{mask(draft.password)}</td></tr>
										</>
									) : (
										<>
											<tr><td>API Key</td><td>{mask(draft.apiKey)}</td></tr>
											<tr><td>API Secret</td><td>{mask(draft.apiSecret)}</td></tr>
											<tr><td>Passphrase</td><td>{mask(draft.passphrase)}</td></tr>
										</>
									)}
									<tr>
										<td>备注</td>
										<td>
											<input className="modal-input" value={draft.note ?? ""} onChange={set("note")}
												autoComplete="off" placeholder="备注（可空）" style={{ width: 220 }} />
										</td>
									</tr>
								</tbody>
							</table>
						)}
						<div style={{ display: "flex", gap: 6, marginTop: 8 }}>
							{step > 1 && <button type="button" className="btn" onClick={back}>上一步</button>}
							{step < maxStep && step !== 3 && (
								<button type="button" className="btn" onClick={next} disabled={!stepFull}>
									下一步
								</button>
							)}
							{step === 3 && (
								<button type="submit" className="btn-primary" disabled={busy}>验证并接入</button>
							)}
						</div>
					</form>
					{step === 4 && kind === "mt5" && (
						<div style={{ marginTop: 10 }}>
							<SymbolMappingStep venue={draft.venue ?? ""} login={draft.login ?? ""}
								loadUniverse={props.loadUniverse}
								onMappingQuery={props.onMappingQuery}
								onSubmitMapping={props.onSubmitMapping} />
						</div>
					)}
				</details>
			)}
			{subErr && <p className="dim">操作未成：{subErr}</p>}
		</Card>
	);
}

// BrokerOnboardCard 容器件（取数+写径装配——ev_pipeline=每拍落行帧重取；
// 写径裸调不入合并窗，提交成功后 reload 同帧直反真值；败由向导步3即回显形）。
export function BrokerOnboardCard() {
	const live = useLive(["ev_pipeline", "ev_strategy"], () => client.listBrokerOnboard({}));
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
			onSubmit={async (f) => {
				await client.submitBrokerCred(f);
				await live.reload();
			}}
			loadUniverse={async () => (await client.listCanonicalUniverse({})).insts}
			onMappingQuery={async (q) => {
				const r = await client.listSymbolCandidates(q);
				return { rows: r.rows, scanned: r.scanned };
			}}
			onSubmitMapping={async (m) => {
				await client.submitSymbolMapping(m);
				await live.reload();
			}}
		/>
	);
}
