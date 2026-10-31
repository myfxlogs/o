package pgstore

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
)

// cmc_tombstone_test.go —— CMC 空仓不落痕票①：墓碑行三锚（空仓落碑 qty=0、
// 连续空仓不重碑、部分平仓只缺腿碑）。

func cmcTestStore(t *testing.T) *Store {
	t.Helper()
	pool := testutil.Pool(t)
	if err := Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return NewStore(pool)
}

func insPos(t *testing.T, s *Store, inst string, qty float64, at time.Time) {
	t.Helper()
	if err := s.InsertCMCPositions(context.Background(), 1, []CMCPosition{{
		InstrumentID: inst, Direction: "LONG", Quantity: qty, EntryPrice: 100,
		EvaluationBidPrice: 99, EvaluationAskPrice: 101, FetchedAt: at,
	}}); err != nil {
		t.Fatal(err)
	}
}

// ①空快照→墓碑落 qty=0；②连续空仓→不重碑；③部分平仓（A平B留）→只A碑。
func TestCMCTombstone(t *testing.T) {
	s := cmcTestStore(t)
	ctx := context.Background()
	now := time.Now()
	insPos(t, s, "X-AFQUL", 89300, now.Add(-time.Hour))
	insPos(t, s, "X-OTHER", 500, now.Add(-time.Hour))

	// ③部分平仓：present={X-OTHER}——只 X-AFQUL 碑。
	n, err := s.TombstoneCMCPositions(ctx, 1, map[string]bool{"X-OTHER": true}, now)
	if err != nil || n != 1 {
		t.Fatalf("部分平仓应只缺腿碑 1: n=%d err=%v", n, err)
	}
	p, err := s.LastCMCPositionByInstrument(ctx, "X-AFQUL")
	if err != nil || p.Quantity != 0 {
		t.Fatalf("A 腿应 qty=0 墓碑: qty=%v err=%v", p.Quantity, err)
	}
	if p.Direction != "LONG" || p.EntryPrice != 100 || p.UnrealisedPnL != 0 || p.MarginAmount != 0 {
		t.Fatalf("墓碑应承末值向/价、零盈亏/占保: %+v", p)
	}
	// B 腿在快照在场——不碑（最新仍 qty=500）。
	b, err := s.LastCMCPositionByInstrument(ctx, "X-OTHER")
	if err != nil || b.Quantity != 500 {
		t.Fatalf("在场腿不应碑: qty=%v", b.Quantity)
	}

	// ②连续空仓：present 空——X-AFQUL 上笔已 qty=0 不重碑；X-OTHER 新缺席 qty>0 落碑。
	n, err = s.TombstoneCMCPositions(ctx, 1, map[string]bool{}, now)
	if err != nil || n != 1 {
		t.Fatalf("连续空仓只 B 腿新碑（A 已碑不重）: n=%d err=%v", n, err)
	}
	var cnt int
	if err := s.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_cmc_positions WHERE instrument_id='X-AFQUL' AND quantity=0").Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("A 腿碑应幂等唯一: %d", cnt)
	}
	b2, err := s.LastCMCPositionByInstrument(ctx, "X-OTHER")
	if err != nil || b2.Quantity != 0 {
		t.Fatalf("B 腿全平后应碑 qty=0: %v", b2.Quantity)
	}
	// ①再拍空快照——双腿皆 qty=0 最新 → 零碑。
	n, err = s.TombstoneCMCPositions(ctx, 1, map[string]bool{}, now)
	if err != nil || n != 0 {
		t.Fatalf("全平后空快照应零碑: n=%d err=%v", n, err)
	}
}
