// 本地 interface 等价件（T-0113：O 无 trx proto 面——字段名对拍 nox gen；
// camelCase 键=api.ts snake→camel 换算件直供）。
import type { Timestamp } from "@bufbuild/protobuf/wkt";

export interface TrxAccountRefreshRequest {}
export interface TrxAccountRefreshResponse {
  id: number; address: string; srAddress: string;
  trxBalance: number; trxAllowance: number;
  energyDelegated: number; bandwidthStaked: number;
  netLimit: number; votes: number; unfrozen: string;
  fetchedAt: Timestamp; netUsed: number;
  freeNetUsed: number; freeNetLimit: number;
  energyLimit: number; energyUsed: number; bandwidthStatus: string;
}
export interface GetTrxYieldRadarRequest {}
export interface GetTrxYieldRadarResponse {
  energyApr7d: number; energyApr30d: number;
  voteAprCurrent: number; voteApr30dAvg: number;
  floorPct: number; sampleDays7d: number; sampleDays30d: number;
  updatedAt: Timestamp; transition: boolean;
  voteAprMax: number; voteAprMaxSr: string; voteAprSpread: number;
}
export interface TrxYieldCurvePoint {
  day: Timestamp; energyTrx: number; votingTrx: number;
  price: number; dailyTotalUsd: number;
}
export interface GetTrxYieldCurveRequest { days: number }
export interface GetTrxYieldCurveResponse {
  points: TrxYieldCurvePoint[]; annualized7d: number; annualized30d: number;
  currentPrice: number; poolPrincipalUsd: number;
  sampleDays7d: number; sampleDays30d: number;
  cumulativeUnhedgedIncomeTrx: number; cumulativeUnhedgedIncomeUsd: number;
}
export interface HedgeStake { energy: number; bandwidth: number }
export interface HedgeLegSpot {
  label: string; qtyStaked: number; qtyLiquid: number;
  pnlVsBase: number; stake: HedgeStake;
  voteAprCurrent: number; voteAprMax: number;
  voteAprMaxSr: string; voteAprSpread: number; voteAprTs: Timestamp;
}
export interface HedgeLegShort {
  label: string; qty: number; pnlVsBase: number; lev: number;
  liq: number; distPct: number; brokerUnrealised: number;
}
export interface HedgeCarry {
  trxRental7dTrx: number; trxVote7dTrx: number;
  cmcOvernight7dUsd: number; cumUsd: number; annualizedPct: number;
  mtdNetUsd: number; mtdSampleDays: number;
  hedgedCumUsd: number; hedgedAnnualizedPct: number;
}
export interface HedgeAccounts {
  trxBalance: number; trxAllowance: number; energyDelegated: number;
  votes: number; cmcCash: number; cmcEquity: number;
  netLimit: number; netUsed: number; freeNetLimit: number; freeNetUsed: number;
  energyLimit: number; energyUsed: number; bandwidthStatus: string;
}
export interface HedgeStructure {
  baselinePrice: number; mid: number;
  legSpot: HedgeLegSpot; legShort: HedgeLegShort;
  netExposureTrx: number; netExposureUsd: number;
  carry: HedgeCarry; accts: HedgeAccounts;
  wsConnected: boolean; closeoutSource: string;
  platformAlertState: string; platformAlerts: string;
  closeoutLevelUsd: number; warningLevelUsd: number; ask: number;
}
export interface GetHedgeStructureRequest {}
export interface GetHedgeStructureResponse { structure: HedgeStructure }
