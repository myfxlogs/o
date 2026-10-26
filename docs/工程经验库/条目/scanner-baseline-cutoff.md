---
schema_version: 2
id: scanner-baseline-cutoff
kind: pitfall
risk: high
status: verified
keywords: [deposit-history, 基线截止, 幂等只防重不防史, venue 全史误灌, ledger 虚增, epoch anchor, 扫描器]
scope: [internal/ledger/funding, 一切 venue 历史回灌扫描面]
summary: venue 历史拉取扫描器必备基线截止谓词——幂等键只防「重复写」不防「首次见到的历史」；无截止=venue 全史被当新事件入账（deposit scan 实案：95 笔前身时代入金 $420,803 误灌=ledger 虚增 8 倍）；锚=账本最早行 occurred_at
first_seen: 2026-10-26
last_verified: 2026-10-26
evidence: [T-0149, SG 生产实证]
---

# scanner-baseline-cutoff——venue 历史扫描器必备基线截止谓词

## 症状
扫描器上线首拍即把 venue 全量历史当新事件灌账——行级幂等全部「首次命中」
正常工作，但语义上把**系统纪元之前的事件**记成了新账。

## 根因
幂等键（bill_id/depId ON CONFLICT）防的是**重复写**，防不了**首次写**——
venue 历史行对扫描器全是「新」的。系统纪元≠venue 纪元：账户在系统之前
已有漫长历史，拉全量=捞前世。

## 实案（T-0149 三返修）
`/asset/deposit-history` 无窗拉取→95 笔前身时代入金（2025-07~2026-09，
$420,803）入账→ledger deposit 面 7k→428k 虚增 8 倍（前身时代真实注资
含已亏/已提部分，与今账无关）。回滚后 bill 行留作「已见未记」证标
天然幂等。

## 修法
`occurred_at ≤ 账基线` → skip（不入账不留源行）+rej 计数。基线锚=
`SELECT min(occurred_at) FROM fin_cashflow`（系统纪元——空账 COALESCE
now() fail-closed）。新事件 postdate 基线=入账面无损。

## 教训
- **一切「回灌/巡检/补漏」扫描器**：问三件事——幂等键（防重）、谓词闸
  （防域外行——本票 ccy 约束毒行实证）、**基线截止（防史）**。三缺一并发。
- venue 拉取有窗则先压窗（时间/条数上限）再叠基线闸——双保险。
- 单行败不可 abort 整扫（毒行饿死后续行）；USDT 域谓词先于写径
  （非注资币种入账=虚增 NAV——D-0021 TRX 自营边界实证）。
