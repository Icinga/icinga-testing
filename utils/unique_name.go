package utils

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var uniqueNameCounters sync.Map

func UniqueName(t testing.TB, hint string) string {
	prefix := t.Name()
	if hint != "" {
		prefix += "-" + hint
	}
	prefix = strings.ReplaceAll(prefix, "/", "-")

	counter, _ := uniqueNameCounters.LoadOrStore(prefix, new(uint32))
	id := atomic.AddUint32(counter.(*uint32), 1)

	return fmt.Sprintf("%s-%d", prefix, id)
}
