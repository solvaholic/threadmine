package utils

import (
	"testing"
	"time"
)

func TestParseSinceDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
		checkFunc   func(t *testing.T, got time.Time)
	}{
		{
			name:    "valid relative format 7d",
			input:   "7d",
			wantErr: false,
			checkFunc: func(t *testing.T, got time.Time) {
				expected := time.Now().AddDate(0, 0, -7)
				// Allow 1 second tolerance for test execution time
				diff := expected.Sub(got)
				if diff > time.Second || diff < -time.Second {
					t.Errorf("expected time around %v, got %v", expected, got)
				}
			},
		},
		{
			name:    "valid relative format 1d",
			input:   "1d",
			wantErr: false,
			checkFunc: func(t *testing.T, got time.Time) {
				expected := time.Now().AddDate(0, 0, -1)
				diff := expected.Sub(got)
				if diff > time.Second || diff < -time.Second {
					t.Errorf("expected time around %v, got %v", expected, got)
				}
			},
		},
		{
			name:    "valid relative format 30d",
			input:   "30d",
			wantErr: false,
			checkFunc: func(t *testing.T, got time.Time) {
				expected := time.Now().AddDate(0, 0, -30)
				diff := expected.Sub(got)
				if diff > time.Second || diff < -time.Second {
					t.Errorf("expected time around %v, got %v", expected, got)
				}
			},
		},
		{
			name:    "valid absolute format",
			input:   "2025-12-15",
			wantErr: false,
			checkFunc: func(t *testing.T, got time.Time) {
				expected := time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)
				if !got.Equal(expected) {
					t.Errorf("expected %v, got %v", expected, got)
				}
			},
		},
		{
			name:    "valid absolute format earlier date",
			input:   "2024-01-01",
			wantErr: false,
			checkFunc: func(t *testing.T, got time.Time) {
				expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				if !got.Equal(expected) {
					t.Errorf("expected %v, got %v", expected, got)
				}
			},
		},
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "invalid format - no number",
			input:       "d",
			wantErr:     true,
			errContains: "invalid relative date format",
		},
		{
			name:        "invalid format - wrong separator",
			input:       "2025/12/15",
			wantErr:     true,
			errContains: "invalid date format",
		},
		{
			name:        "invalid format - incomplete date",
			input:       "2025-12",
			wantErr:     true,
			errContains: "invalid date format",
		},
		{
			name:        "invalid format - not a date",
			input:       "yesterday",
			wantErr:     true,
			errContains: "invalid date format",
		},
		{
			name:        "negative days",
			input:       "-7d",
			wantErr:     true,
			errContains: "days cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSinceDate(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseSinceDate() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ParseSinceDate() error = %v, should contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseSinceDate() unexpected error = %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
