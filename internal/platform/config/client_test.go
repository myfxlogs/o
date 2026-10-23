package config

import (
	"context"
	"os"
	"testing"
	"time"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

func TestFailClosedBeforeLoad(t *testing.T) {
	pool := testutil.Pool(t)
	c := New(pool)
	if _, err := c.Get("anything"); err == nil {
		t.Fatal("未就绪 Get 应 fail-closed")
	}
}

func TestLoadGetNotify(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO app_config(key,value) VALUES('test.k','\"v1\"') ON CONFLICT(key) DO UPDATE SET value='\"v1\"'"); err != nil {
		t.Fatal(err)
	}
	c := New(pool)
	if err := c.Load(ctx); err != nil {
		t.Fatalf("load: %v", err)
	}
	v, err := c.Get("test.k")
	if err != nil || string(v) != `"v1"` {
		t.Fatalf("get: v=%s err=%v", v, err)
	}
	if _, err := c.Get("missing.key"); err == nil {
		t.Fatal("缺键应 fail-closed")
	}

	wctx, cancel := context.WithCancel(ctx)
	defer cancel()
	watchErr := make(chan error, 1)
	go func() { watchErr <- c.Watch(wctx) }()
	time.Sleep(300 * time.Millisecond) // 等 LISTEN 上耳
	if _, err := pool.Exec(ctx,
		"UPDATE app_config SET value='\"v2\"' WHERE key='test.k'"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		v, _ := c.Get("test.k")
		if string(v) == `"v2"` {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("notify 重载未生效 gen=%d", c.Generation())
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestMasterKeyFile(t *testing.T) {
	t.Setenv("PHASE_MASTER_KEY_FILE", "/nonexistent/path/key")
	if _, err := ReadMasterKey(); err == nil {
		t.Fatal("缺件应 fail-closed")
	}
	f := t.TempDir() + "/k"
	if err := os.WriteFile(f, []byte("testkey-material-32B-padded-000"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PHASE_MASTER_KEY_FILE", f)
	b, err := ReadMasterKey()
	if err != nil || len(b) == 0 {
		t.Fatalf("read: %v len=%d", err, len(b))
	}
}
