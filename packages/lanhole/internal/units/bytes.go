package units

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseBytes(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "0" || s == "none" || s == "unlimited" {
		return 0, nil
	}
	multipliers := []struct {
		suffix string
		mul    int64
	}{
		{"kib", 1 << 10}, {"kb", 1000}, {"k", 1000},
		{"mib", 1 << 20}, {"mb", 1000 * 1000}, {"m", 1000 * 1000},
		{"gib", 1 << 30}, {"gb", 1000 * 1000 * 1000}, {"g", 1000 * 1000 * 1000},
		{"tib", 1 << 40}, {"tb", 1000 * 1000 * 1000 * 1000}, {"t", 1000 * 1000 * 1000 * 1000},
		{"b", 1},
	}
	mul := int64(1)
	num := s
	for _, m := range multipliers {
		if strings.HasSuffix(s, m.suffix) {
			mul = m.mul
			num = strings.TrimSpace(strings.TrimSuffix(s, m.suffix))
			break
		}
	}
	if num == "" {
		return 0, fmt.Errorf("bad size %q", s)
	}
	v, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, fmt.Errorf("bad size %q: %w", s, err)
	}
	if v < 0 {
		return 0, fmt.Errorf("bad negative size %q", s)
	}
	return int64(v * float64(mul)), nil
}

func FormatBytes(n int64) string {
	if n == 0 {
		return "unlimited"
	}
	const gib = int64(1 << 30)
	const mib = int64(1 << 20)
	const kib = int64(1 << 10)
	switch {
	case n%gib == 0 && n >= gib:
		return fmt.Sprintf("%dGiB", n/gib)
	case n%mib == 0 && n >= mib:
		return fmt.Sprintf("%dMiB", n/mib)
	case n%kib == 0 && n >= kib:
		return fmt.Sprintf("%dKiB", n/kib)
	default:
		return fmt.Sprintf("%dB", n)
	}
}
