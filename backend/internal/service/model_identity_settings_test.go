package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type identitySettingsRepo struct {
	SettingRepository
	value string
}

func (r *identitySettingsRepo) GetValue(context.Context, string) (string, error) {
	if r.value == "" {
		return "", ErrSettingNotFound
	}
	return r.value, nil
}
func (r *identitySettingsRepo) Set(_ context.Context, key, value string) error {
	if key != identityPublicURLSetting {
		panic("unexpected settings key")
	}
	r.value = value
	return nil
}

func TestIdentitySettingsPersistBaseURLAndIgnoreEnvironment(t *testing.T) {
	t.Setenv("MODEL_IDENTITY_PUBLIC_BASE_URL", "https://obsolete.example.com")
	repo := &identitySettingsRepo{}
	svc := NewModelIdentityService(nil, nil, nil, repo)
	ctx := context.Background()
	settings, err := svc.Settings(ctx)
	require.NoError(t, err)
	require.Empty(t, settings.PublicBaseURL)
	_, err = svc.publicBaseURL(ctx)
	require.ErrorContains(t, err, "Model identity page")
	settings, err = svc.SaveSettings(ctx, " https://site.example.com/prefix/ ")
	require.NoError(t, err)
	require.Equal(t, "https://site.example.com/prefix", settings.PublicBaseURL)
	restarted := NewModelIdentityService(nil, nil, nil, repo)
	value, err := restarted.publicBaseURL(ctx)
	require.NoError(t, err)
	require.Equal(t, settings.PublicBaseURL, value)
	for _, invalid := range []string{"", "http://site.example.com", "https://user:password@site.example.com", "https://site.example.com?key=secret", "https://site.example.com?", "https://site.example.com#fragment", "https://localhost", "https://127.0.0.1", "https://10.0.0.1", "https://[::1]"} {
		_, err := svc.SaveSettings(ctx, invalid)
		require.Error(t, err, invalid)
		require.Equal(t, value, repo.value, "invalid settings must not replace the saved URL")
	}
}
