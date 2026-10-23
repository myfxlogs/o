package cryptoutil

import "testing"

func TestSealOpenRoundtrip(t *testing.T) {
	key := []byte("passphrase-material")
	b, err := Seal(key, "plain-42")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Open(key, b)
	if err != nil || got != "plain-42" {
		t.Fatalf("roundtrip: %q %v", got, err)
	}
	if _, err := Open([]byte("wrong-key"), b); err == nil {
		t.Fatal("错 key 应拒解")
	}
	b2, _ := Seal(key, "plain-42")
	if string(b) == string(b2) {
		t.Fatal("nonce not random")
	}
}
