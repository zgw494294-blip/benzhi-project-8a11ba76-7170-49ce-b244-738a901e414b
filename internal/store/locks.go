package store

import (
	"hash/fnv"
	"sync"
)

const lockShardCount = 64

type caseLocks struct {
	shards [lockShardCount]sync.Mutex
}

func (l *caseLocks) forCase(caseID string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(caseID))
	return &l.shards[h.Sum32()%lockShardCount]
}
