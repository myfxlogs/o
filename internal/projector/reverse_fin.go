// reverse_fin.go —— T-0326 返修①五 fin 表升 bridge-reverse 副本面（stakeCarryScan
// 质押腿断供根修；reverse.go 450 行硬线分拆——同包同 Runner，机制=revTableDiff
// 泛型双侧全表 diff，照 reverseStrategies/reverseEarnRate 形勿造新范式）。
package projector

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// T-0326 返修①：stakeCarryScan 质押腿断供根修——五 fin 表升 bridge-reverse
// （升件1593 补裁①：md_1070 先例同形，SG 正本→PG-md 只读副本全表 diff；
// 体量全小 vote_apr 2409/trx_account 3008/trx_txs 813/cmc_history 39/
// cmc_positions 8390——仅差行过隧道）。无 SG 触发帧——ev_strategy 帧顺带
// +Reconcile 期界对账兜底（勿造新 NOTIFY 范式）。消费面：trxOnchainYield
// vote 年化+rental 分子/分母（fin_trx_vote_apr/fin_trx_txs/fin_trx_account）、
// shortCarryAnn cmc_realized 结算/空腿名义（fin_cmc_history/fin_cmc_positions）。
// numeric 以文本搬运保精度（mirror.go 同律）；行判等=reflect.DeepEqual——
// 指针列按所指值比（h!=s 结构体判等含指针列恒不等=全表恒覆写浪费，同形
// 先例如实修）。
const qTrxVoteAprRev = `SELECT id,fetched_at,current_rate,current_sr,max_rate,max_sr,
	spread,min_rate,min_sr FROM fin_trx_vote_apr`

const uTrxVoteAprMirror = `INSERT INTO fin_trx_vote_apr(id,fetched_at,current_rate,
	current_sr,max_rate,max_sr,spread,min_rate,min_sr)
	VALUES($1,$2::timestamptz,$3,$4,$5,$6,$7,$8,$9)
	ON CONFLICT (id) DO UPDATE SET fetched_at=EXCLUDED.fetched_at,
		current_rate=EXCLUDED.current_rate,current_sr=EXCLUDED.current_sr,
		max_rate=EXCLUDED.max_rate,max_sr=EXCLUDED.max_sr,spread=EXCLUDED.spread,
		min_rate=EXCLUDED.min_rate,min_sr=EXCLUDED.min_sr`

const qTrxAccountRev = `SELECT id,address,votes_total,sr_address,balance_sun,
	allowance_sun,energy_delegated,bandwidth_staked,net_limit,unfrozen::text,
	fetched_at,tron_power,net_used,free_net_used,free_net_limit,energy_limit,
	energy_used,reward_num_sun FROM fin_trx_account`

const uTrxAccountMirror = `INSERT INTO fin_trx_account(id,address,votes_total,
	sr_address,balance_sun,allowance_sun,energy_delegated,bandwidth_staked,
	net_limit,unfrozen,fetched_at,tron_power,net_used,free_net_used,
	free_net_limit,energy_limit,energy_used,reward_num_sun)
	VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11::timestamptz,$12,$13,$14,$15,$16,$17,$18)
	ON CONFLICT (id) DO UPDATE SET address=EXCLUDED.address,
		votes_total=EXCLUDED.votes_total,sr_address=EXCLUDED.sr_address,
		balance_sun=EXCLUDED.balance_sun,allowance_sun=EXCLUDED.allowance_sun,
		energy_delegated=EXCLUDED.energy_delegated,bandwidth_staked=EXCLUDED.bandwidth_staked,
		net_limit=EXCLUDED.net_limit,unfrozen=EXCLUDED.unfrozen,fetched_at=EXCLUDED.fetched_at,
		tron_power=EXCLUDED.tron_power,net_used=EXCLUDED.net_used,
		free_net_used=EXCLUDED.free_net_used,free_net_limit=EXCLUDED.free_net_limit,
		energy_limit=EXCLUDED.energy_limit,energy_used=EXCLUDED.energy_used,
		reward_num_sun=EXCLUDED.reward_num_sun`

const qTrxTxsRev = `SELECT id,tx_id,ts,kind,amount_sun,counterparty,fetched_at,
	cheat_status FROM fin_trx_txs`

const uTrxTxsMirror = `INSERT INTO fin_trx_txs(id,tx_id,ts,kind,amount_sun,
	counterparty,fetched_at,cheat_status)
	VALUES($1,$2,$3::timestamptz,$4,$5,$6,$7::timestamptz,$8)
	ON CONFLICT (id) DO UPDATE SET tx_id=EXCLUDED.tx_id,ts=EXCLUDED.ts,
		kind=EXCLUDED.kind,amount_sun=EXCLUDED.amount_sun,
		counterparty=EXCLUDED.counterparty,fetched_at=EXCLUDED.fetched_at,
		cheat_status=EXCLUDED.cheat_status`

const qCmcHistoryRev = `SELECT id,account_id,transaction_id,transaction_type,
	amount::text,balance::text,currency,order_id,transaction_time,fetched_at
	FROM fin_cmc_history`

const uCmcHistoryMirror = `INSERT INTO fin_cmc_history(id,account_id,transaction_id,
	transaction_type,amount,balance,currency,order_id,transaction_time,fetched_at)
	VALUES($1,$2,$3,$4,$5::numeric,$6::numeric,$7,$8,$9::timestamptz,$10::timestamptz)
	ON CONFLICT (id) DO UPDATE SET account_id=EXCLUDED.account_id,
		transaction_id=EXCLUDED.transaction_id,transaction_type=EXCLUDED.transaction_type,
		amount=EXCLUDED.amount,balance=EXCLUDED.balance,currency=EXCLUDED.currency,
		order_id=EXCLUDED.order_id,transaction_time=EXCLUDED.transaction_time,
		fetched_at=EXCLUDED.fetched_at`

const qCmcPositionsRev = `SELECT id,account_id,instrument_id,direction,
	quantity::text,entry_price::text,unrealised_pnl::text,margin_amount::text,
	margin_percentage::text,evaluation_bid_price::text,evaluation_ask_price::text,
	opened_at,order_id,fetched_at FROM fin_cmc_positions`

const uCmcPositionsMirror = `INSERT INTO fin_cmc_positions(id,account_id,instrument_id,
	direction,quantity,entry_price,unrealised_pnl,margin_amount,margin_percentage,
	evaluation_bid_price,evaluation_ask_price,opened_at,order_id,fetched_at)
	VALUES($1,$2,$3,$4,$5::numeric,$6::numeric,$7::numeric,$8::numeric,$9::numeric,
		$10::numeric,$11::numeric,$12::timestamptz,$13,$14::timestamptz)
	ON CONFLICT (id) DO UPDATE SET account_id=EXCLUDED.account_id,
		instrument_id=EXCLUDED.instrument_id,direction=EXCLUDED.direction,
		quantity=EXCLUDED.quantity,entry_price=EXCLUDED.entry_price,
		unrealised_pnl=EXCLUDED.unrealised_pnl,margin_amount=EXCLUDED.margin_amount,
		margin_percentage=EXCLUDED.margin_percentage,
		evaluation_bid_price=EXCLUDED.evaluation_bid_price,
		evaluation_ask_price=EXCLUDED.evaluation_ask_price,opened_at=EXCLUDED.opened_at,
		order_id=EXCLUDED.order_id,fetched_at=EXCLUDED.fetched_at`

// revTableDiff 五 fin 表同构双侧全表 diff（T-0326）——want=SG 正本、have=md
// 副本，行判等 reflect.DeepEqual（指针列按所指值比——h!=s 结构体判等含指针
// 列恒不等=全表恒覆写）；缺/差行进 md upsert 批（写权唯一=本臂）。
func revTableDiff[T any](ctx context.Context, sg, md *pgxpool.Pool, q string,
	scan func(pgx.Rows) (int64, T, []any, error), upsert string) error {
	fetch := func(pool *pgxpool.Pool) (map[int64]T, map[int64][]any, error) {
		rows, err := pool.Query(ctx, q)
		if err != nil {
			return nil, nil, err
		}
		defer rows.Close()
		m := map[int64]T{}
		args := map[int64][]any{}
		for rows.Next() {
			id, row, a, err := scan(rows)
			if err != nil {
				return nil, nil, err
			}
			m[id] = row
			args[id] = a
		}
		return m, args, rows.Err()
	}
	want, wargs, err := fetch(sg)
	if err != nil {
		return fmt.Errorf("正本读: %w", err)
	}
	have, _, err := fetch(md)
	if err != nil {
		return fmt.Errorf("副本读: %w", err)
	}
	var batch [][]any
	for id, w := range want {
		if h, ok := have[id]; !ok || !reflect.DeepEqual(h, w) {
			batch = append(batch, wargs[id])
		}
	}
	return batchExec(ctx, md, upsert, batch)
}

type trxVoteAprRow struct {
	fetchedAt       time.Time
	curRate         float64
	curSR           *string
	maxRate         float64
	maxSR           *string
	spread, minRate float64
	minSR           *string
}

// reverseTrxVoteApr 投票年化正本→副本（SpotYield vote 分量读面——T-0326）。
func (r *Runner) reverseTrxVoteApr(ctx context.Context) error {
	return revTableDiff(ctx, r.sg, r.md, qTrxVoteAprRev,
		func(rows pgx.Rows) (int64, trxVoteAprRow, []any, error) {
			var id int64
			var s trxVoteAprRow
			err := rows.Scan(&id, &s.fetchedAt, &s.curRate, &s.curSR, &s.maxRate,
				&s.maxSR, &s.spread, &s.minRate, &s.minSR)
			return id, s, []any{id, s.fetchedAt, s.curRate, s.curSR, s.maxRate,
				s.maxSR, s.spread, s.minRate, s.minSR}, err
		}, uTrxVoteAprMirror)
}

type trxAccountRow struct {
	address                                       string
	votesTotal                                    int64
	srAddress                                     *string
	balanceSun, allowanceSun                      int64
	energyDelegated, bandwidthStaked, netLimit    int64
	unfrozen                                      *string
	fetchedAt                                     time.Time
	tronPower, netUsed, freeNetUsed, freeNetLimit int64
	energyLimit, energyUsed, rewardNumSun         int64
}

// reverseTrxAccount 质押账户正本→副本（rental 分母 energy_delegated 读面——T-0326）。
func (r *Runner) reverseTrxAccount(ctx context.Context) error {
	return revTableDiff(ctx, r.sg, r.md, qTrxAccountRev,
		func(rows pgx.Rows) (int64, trxAccountRow, []any, error) {
			var id int64
			var s trxAccountRow
			err := rows.Scan(&id, &s.address, &s.votesTotal, &s.srAddress,
				&s.balanceSun, &s.allowanceSun, &s.energyDelegated, &s.bandwidthStaked,
				&s.netLimit, &s.unfrozen, &s.fetchedAt, &s.tronPower, &s.netUsed,
				&s.freeNetUsed, &s.freeNetLimit, &s.energyLimit, &s.energyUsed,
				&s.rewardNumSun)
			return id, s, []any{id, s.address, s.votesTotal, s.srAddress,
				s.balanceSun, s.allowanceSun, s.energyDelegated, s.bandwidthStaked,
				s.netLimit, s.unfrozen, s.fetchedAt, s.tronPower, s.netUsed,
				s.freeNetUsed, s.freeNetLimit, s.energyLimit, s.energyUsed,
				s.rewardNumSun}, err
		}, uTrxAccountMirror)
}

type trxTxsRow struct {
	txID         string
	ts           time.Time
	kind         string
	amountSun    int64
	counterparty *string
	fetchedAt    time.Time
	cheatStatus  bool
}

// reverseTrxTxs 链上流水正本→副本（rental_income 分子读面——T-0326）。
func (r *Runner) reverseTrxTxs(ctx context.Context) error {
	return revTableDiff(ctx, r.sg, r.md, qTrxTxsRev,
		func(rows pgx.Rows) (int64, trxTxsRow, []any, error) {
			var id int64
			var s trxTxsRow
			err := rows.Scan(&id, &s.txID, &s.ts, &s.kind, &s.amountSun,
				&s.counterparty, &s.fetchedAt, &s.cheatStatus)
			return id, s, []any{id, s.txID, s.ts, s.kind, s.amountSun,
				s.counterparty, s.fetchedAt, s.cheatStatus}, err
		}, uTrxTxsMirror)
}

type cmcHistoryRow struct {
	accountID         int64
	txID, txType      *string
	amount, balance   *string
	currency, orderID *string
	txTime, fetchedAt *time.Time
}

// reverseCmcHistory 结算流水正本→副本（cmc_realized 隔夜结算读面——T-0326）。
func (r *Runner) reverseCmcHistory(ctx context.Context) error {
	return revTableDiff(ctx, r.sg, r.md, qCmcHistoryRev,
		func(rows pgx.Rows) (int64, cmcHistoryRow, []any, error) {
			var id int64
			var s cmcHistoryRow
			err := rows.Scan(&id, &s.accountID, &s.txID, &s.txType, &s.amount,
				&s.balance, &s.currency, &s.orderID, &s.txTime, &s.fetchedAt)
			return id, s, []any{id, s.accountID, s.txID, s.txType, s.amount,
				s.balance, s.currency, s.orderID, s.txTime, s.fetchedAt}, err
		}, uCmcHistoryMirror)
}

type cmcPositionsRow struct {
	accountID            int64
	instID, direction    *string
	quantity, entryPrice *string
	unrealised, margin   *string
	marginPct            *string
	evalBid, evalAsk     *string
	openedAt             *time.Time
	orderID              *string
	fetchedAt            *time.Time
}

// reverseCmcPositions 持仓面正本→副本（shortCarryAnn 空腿名义读面——T-0326）。
func (r *Runner) reverseCmcPositions(ctx context.Context) error {
	return revTableDiff(ctx, r.sg, r.md, qCmcPositionsRev,
		func(rows pgx.Rows) (int64, cmcPositionsRow, []any, error) {
			var id int64
			var s cmcPositionsRow
			err := rows.Scan(&id, &s.accountID, &s.instID, &s.direction,
				&s.quantity, &s.entryPrice, &s.unrealised, &s.margin, &s.marginPct,
				&s.evalBid, &s.evalAsk, &s.openedAt, &s.orderID, &s.fetchedAt)
			return id, s, []any{id, s.accountID, s.instID, s.direction,
				s.quantity, s.entryPrice, s.unrealised, s.margin, s.marginPct,
				s.evalBid, s.evalAsk, s.openedAt, s.orderID, s.fetchedAt}, err
		}, uCmcPositionsMirror)
}

// reverseFinMirrors 五 fin 表副本群 diff 汇总径（T-0326——ev_strategy 帧顺带
// +Reconcile 期界对账同口调用；无 SG 触发帧不造新 NOTIFY 范式）。
func (r *Runner) reverseFinMirrors(ctx context.Context) error {
	for _, fn := range []struct {
		name string
		f    func(context.Context) error
	}{
		{"trx_vote_apr", r.reverseTrxVoteApr},
		{"trx_account", r.reverseTrxAccount},
		{"trx_txs", r.reverseTrxTxs},
		{"cmc_history", r.reverseCmcHistory},
		{"cmc_positions", r.reverseCmcPositions},
	} {
		if err := fn.f(ctx); err != nil {
			return fmt.Errorf("%s: %w", fn.name, err)
		}
	}
	return nil
}
