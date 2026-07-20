package transform

import (
	"context"
	"testing"
)

// benchData builds a decoded JSON-like value with n records, the shape these query
// engines actually receive after Terraform decodes the input.
func benchData(n int) interface{} {
	records := make([]interface{}, n)
	for i := 0; i < n; i++ {
		records[i] = map[string]interface{}{
			"name":    "service-" + itoa(i),
			"port":    float64(8000 + i),
			"enabled": i%2 == 0,
			"tags":    []interface{}{"prod", "web", "region-" + itoa(i%5)},
		}
	}
	return map[string]interface{}{"records": records}
}

func BenchmarkJQ(b *testing.B) {
	ctx := context.Background()
	for _, n := range []int{10, 100, 1000} {
		data := benchData(n)
		b.Run(itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := runJQ(ctx, data, `.records[] | select(.enabled) | .name`, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkJMESPath(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		data := benchData(n)
		b.Run(itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := runJMESPath(data, `records[?enabled].name`); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkJSONata(b *testing.B) {
	ctx := context.Background()
	for _, n := range []int{10, 100, 1000} {
		data := benchData(n)
		b.Run(itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := runJSONata(ctx, data, `records[enabled].name`); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
