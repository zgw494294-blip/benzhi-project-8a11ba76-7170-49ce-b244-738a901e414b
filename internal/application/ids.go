package application

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func newID(prefix string) (string, error) {
	var random [10]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("生成标识: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(random[:]), nil
}

func stableID(prefix, namespace string) string {
	digest := sha256.Sum256([]byte(namespace))
	return prefix + "_" + hex.EncodeToString(digest[:10])
}
