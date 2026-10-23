package config

import (
	"context"
	"testing"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

func TestTypedReaders(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 原生类型种子（裸数字+原生布尔+字符串）
	for _, q := range []string{
		`INSERT INTO app_config(key,value) VALUES('t.f','5.5'::jsonb),('t.b','true'::jsonb),('t.s','"abc"'::jsonb),('t.fs','"7.7"'::jsonb),('t.bs','"false"'::jsonb)
		 ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	c := New(pool)
	if err := c.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if v, _ := c.GetFloat("t.f"); v != 5.5 {
		t.Fatalf("native float=%v", v)
	}
	if v, _ := c.GetFloat("t.fs"); v != 7.7 {
		t.Fatalf("string-coded float（宽容两形）=%v", v)
	}
	if v, _ := c.GetBool("t.b"); !v {
		t.Fatal("native bool")
	}
	if v, _ := c.GetBool("t.bs"); v {
		t.Fatal("string bool")
	}
	if v, _ := c.GetString("t.s"); v != "abc" {
		t.Fatal("string")
	}
	// 类型不符 fail-closed 锚
	if _, err := c.GetFloat("t.s"); err == nil {
		t.Fatal("字符串键读 float 应 fail-closed")
	}
	if _, err := c.GetBool("t.f"); err == nil {
		t.Fatal("数字键读 bool 应 fail-closed")
	}
}
