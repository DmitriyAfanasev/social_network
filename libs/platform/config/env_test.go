package config

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnv(t *testing.T) {
	t.Setenv("CONFIG_TEST_VALUE", "configured")

	if got := Env("CONFIG_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("Env() = %q, want %q", got, "configured")
	}
	if got := Env("CONFIG_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("Env() for missing value = %q, want %q", got, "fallback")
	}
}

func TestIntegerParsers(t *testing.T) {
	tests := []struct {
		name     string
		parse    func(string, int) int
		value    string
		fallback int
		want     int
	}{
		{name: "positive", parse: PositiveInt, value: "7", fallback: 3, want: 7},
		{name: "positive rejects zero", parse: PositiveInt, value: "0", fallback: 3, want: 3},
		{name: "non negative accepts zero", parse: NonNegativeInt, value: "0", fallback: 3, want: 0},
		{name: "invalid", parse: PositiveInt, value: "invalid", fallback: 3, want: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.parse(test.value, test.fallback); got != test.want {
				t.Fatalf("parser(%q, %d) = %d, want %d", test.value, test.fallback, got, test.want)
			}
		})
	}
}

func TestDurationParsers(t *testing.T) {
	if got := PositiveDuration("2s", time.Second); got != 2*time.Second {
		t.Fatalf("PositiveDuration() = %s, want 2s", got)
	}
	if got := PositiveDuration("0s", time.Second); got != time.Second {
		t.Fatalf("PositiveDuration() for zero = %s, want 1s", got)
	}
	if got := NonNegativeDuration("0s", time.Second); got != 0 {
		t.Fatalf("NonNegativeDuration() = %s, want 0", got)
	}
	if got := SecondsDuration("30", time.Second); got != 30*time.Second {
		t.Fatalf("SecondsDuration() = %s, want 30s", got)
	}
}

func TestLogLevel(t *testing.T) {
	if got := LogLevel("warning"); got != slog.LevelWarn {
		t.Fatalf("LogLevel() = %s, want WARN", got)
	}
	if got := LogLevel("unknown"); got != slog.LevelInfo {
		t.Fatalf("LogLevel() for unknown value = %s, want INFO", got)
	}
}

func TestNumericAndDurationFallbacks(t *testing.T) {
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"positive invalid", PositiveInt("nope", 7), 7},
		{"positive zero", PositiveInt("0", 7), 7},
		{"nonnegative valid zero", NonNegativeInt("0", 7), 0},
		{"nonnegative negative", NonNegativeInt("-1", 7), 7},
		{"positive duration", PositiveDuration("250ms", time.Second), 250 * time.Millisecond},
		{"positive duration invalid", PositiveDuration("0s", time.Second), time.Second},
		{"nonnegative duration zero", NonNegativeDuration("0s", time.Second), time.Duration(0)},
		{"seconds", SecondsDuration("12", time.Second), 12 * time.Second},
		{"seconds invalid", SecondsDuration("-1", time.Second), time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { require.Equal(t, tt.want, tt.got) })
	}
}
