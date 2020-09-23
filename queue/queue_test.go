package queue

import (
	"math/rand"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
)

type Queue interface {
	Enqueue(value interface{})
	Dequeue() interface{}
}

func TestQueue(t *testing.T) {
	queues := map[string]Queue{
		"lock-free-queue": NewLFQueue(),
		"lock-queue":      NewLKQueue(0),
	}

	for name, q := range queues {
		t.Run(name, func(t *testing.T) {
			count := 100
			for i := 0; i < count; i++ {
				q.Enqueue(i)
			}

			for i := 0; i < count; i++ {
				v := q.Dequeue()
				if v == nil {
					t.Fatalf("got a nil value")
				}
				if v.(int) != i {
					t.Fatalf("expect %d but got %v", i, v)
				}
			}
		})
	}

}

func BenchmarkQueue(b *testing.B) {
	queues := map[string]Queue{
		"lock-free-queue": NewLFQueue(),
		"lock-queue":      NewLKQueue(0),
	}

	length := 1 << 12
	inputs := make([]int, length)
	for i := 0; i < length; i++ {
		inputs = append(inputs, rand.Int())
	}

	for _, cpus := range []int{4, 8, 16, 32, 64} {
		runtime.GOMAXPROCS(cpus)
		for name, q := range queues {
			b.Run(name+"#"+strconv.Itoa(cpus), func(b *testing.B) {
				b.ResetTimer()

				var c int64
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						i := int(atomic.AddInt64(&c, 1)-1) % length
						v := inputs[i]
						if v >= 0 {
							q.Enqueue(v)
						} else {
							q.Dequeue()
						}
					}
				})
			})
		}
	}
}
