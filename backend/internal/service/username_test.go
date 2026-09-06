//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAndValidateUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "trims surrounding whitespace", input: "  Alice  ", want: "Alice"},
		{name: "rejects empty", input: "   ", wantErr: ErrUsernameRequired},
		{name: "rejects control characters", input: "alice\nname", wantErr: ErrUsernameInvalid},
		{name: "rejects over one hundred runes", input: strings.Repeat("a", MaxUsernameRunes+1), wantErr: ErrUsernameInvalid},
		{name: "accepts exactly one hundred runes", input: strings.Repeat("a", MaxUsernameRunes), want: strings.Repeat("a", MaxUsernameRunes)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeAndValidateUsername(tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
