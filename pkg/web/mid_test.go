package web

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func BenchmarkMap(b *testing.B) {
	b.Run("ip limiter", func(b *testing.B) {
		r := rand.New(rand.NewSource(1))
		ipRateLimiter := IDRateLimiter(10, 10, 3*time.Minute)
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ipRateLimiter(fmt.Sprintf("127.0.0.%v", r.Intn(100)+1))
			}
		})
	})

	b.Run("load or store", func(b *testing.B) {
		r := rand.New(rand.NewSource(1))
		var a sync.Map
		for i := 0; i < b.N; i++ {
			a.LoadOrStore(r.Intn(100), rate.NewLimiter(10, 10))
		}
	})

	b.Run("load", func(b *testing.B) {
		r := rand.New(rand.NewSource(1))
		var a sync.Map
		for i := 0; i < b.N; i++ {
			k := r.Intn(100)
			_, ok := a.Load(k)
			if !ok {
				a.Store(k, rate.NewLimiter(10, 10))
			}
		}
	})
}
