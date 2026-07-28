package unmask

import "testing"

func BenchmarkSkeleton(b *testing.B) {
	for _, s := range []string{"paypal", "pаypаl", "verylongbrandname"} {
		b.Run(s, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = Skeleton(s)
			}
		})
	}
}

func BenchmarkAnalyze(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Analyze("pаypаl")
	}
}
