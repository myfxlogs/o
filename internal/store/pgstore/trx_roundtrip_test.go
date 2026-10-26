package pgstore

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
)

// T-0108 桩测锚：trx 存储往返（InsertTRXAccount→LastTRXAccount 读回同值——
// schema 落库对拍；DB 测按 O_TEST_PG_DSN 实跑约定）。
func TestTrxStoreRoundtrip(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	s := NewStore(pool)
	sr := "SR1"
	if err := s.InsertTRXAccount(ctx, TRXAccount{
		Address: "TADDR", VotesTotal: 100, SRAddress: &sr, BalanceSun: 1234567,
		TronPower: 500000, RewardNumSun: 42}); err != nil {
		t.Fatal(err)
	}
	got, err := s.LastTRXAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Address != "TADDR" || got.BalanceSun != 1234567 || got.RewardNumSun != 42 ||
		got.SRAddress == nil || *got.SRAddress != "SR1" {
		t.Fatalf("往返错位: %+v", got)
	}
}
