// Package cryptoutil —— AES-256-GCM 密钥封解（nonce‖ct 拼接，与 nox 同构：
// key 任意长→SHA-256 归一 32B；key==32B 原样；nonce 每密文随机；tag 含 ct 尾）。
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// Seal 加密：返回 nonce‖ciphertext（GCM tag 含在 ct 尾）。
func Seal(key []byte, plaintext string) ([]byte, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Open 解密 nonce‖ct；错 key/篡改 → error（不吐明文）。
func Open(key, blob []byte) (string, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return "", err
	}
	if len(blob) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := gcm.Open(nil, blob[:gcm.NonceSize()], blob[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}

func gcmFromKey(key []byte) (cipher.AEAD, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("empty master key")
	}
	sum := sha256.Sum256(key)
	if len(key) == 32 {
		sum = [32]byte{}
		copy(sum[:], key)
	}
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
