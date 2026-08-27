package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash"
)

var aggregateDigest hash.Hash = sha256.New()
var objectDigest hash.Hash = sha256.New()

func Digest(v any) string {
	b, _ := json.Marshal(v)
	aggregateDigest.Reset()
	_, _ = aggregateDigest.Write(b)
	return hex.EncodeToString(aggregateDigest.Sum(nil))
}

func DigestBytes(b []byte) string {
	objectDigest.Reset()
	_, _ = objectDigest.Write(b)
	return hex.EncodeToString(objectDigest.Sum(nil))
}
