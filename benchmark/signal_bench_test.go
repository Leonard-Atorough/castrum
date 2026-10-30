package benchmark

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/signal"
)

type benchSignal struct{ n int }

var benchSink int

// BenchmarkHubEmitRead reports one fixed-tick window: n emits followed by
// two independent readers reading all of them, then the tick rotation -
// the per-signal cost of the buffered design (map lookup and append on
// emit, cursor walk and handler call on drain).
func BenchmarkHubEmitRead(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(intName(n), func(b *testing.B) {
			hub := signal.NewHub()
			hub.BeginFrame()
			hub.EndFrame()
			first := hub.NewReader[benchSignal]()
			second := hub.NewReader[benchSignal]()

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for i := range n {
					hub.Emit(benchSignal{n: i})
				}
				for _, r := range []*signal.Reader[benchSignal]{first, second} {
					for value, ok := r.Next(); ok; value, ok = r.Next() {
						benchSink += value.n
					}
				}
				hub.EndTick()
			}
		})
	}
}
