package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type identitySettingsRepo struct {
	SettingRepository
	values            map[string]string
	readErr, writeErr error
}

func (r *identitySettingsRepo) GetValue(_ context.Context, key string) (string, error) {
	if r.readErr != nil {
		return "", r.readErr
	}
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}
func (r *identitySettingsRepo) Set(_ context.Context, key, value string) error {
	if r.writeErr != nil {
		return r.writeErr
	}
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func TestIdentitySettingsPersistAndIgnoreEnvironment(t *testing.T) {
	t.Setenv("MODEL_IDENTITY_PUBLIC_BASE_URL", "https://obsolete.example.com")
	repo := &identitySettingsRepo{}
	svc := NewModelIdentityService(nil, nil, nil, repo)
	ctx := context.Background()
	settings, err := svc.Settings(ctx)
	require.NoError(t, err)
	require.Equal(t, DefaultIdentitySettings(), *settings)
	_, err = svc.publicBaseURL(ctx)
	require.ErrorContains(t, err, "Model identity page")
	settings, err = svc.SaveSettings(ctx, IdentitySettings{PublicBaseURL: " https://site.example.com/prefix/ ", ConcurrentAccounts: 2, ProbeConcurrency: 7})
	require.NoError(t, err)
	require.Equal(t, "https://site.example.com/prefix", settings.PublicBaseURL)
	restarted := NewModelIdentityService(nil, nil, nil, repo)
	loaded, err := restarted.Settings(ctx)
	require.NoError(t, err)
	require.Equal(t, settings, loaded)
	value, err := restarted.publicBaseURL(ctx)
	require.NoError(t, err)
	require.Equal(t, settings.PublicBaseURL, value)
	original := repo.values[identitySettingsSetting]
	for _, invalid := range []string{"", "http://site.example.com", "https://user:password@site.example.com", "https://site.example.com?key=secret", "https://site.example.com?", "https://site.example.com#fragment", "https://localhost", "https://127.0.0.1", "https://10.0.0.1", "https://[::1]"} {
		candidate := *settings
		candidate.PublicBaseURL = invalid
		_, err := svc.SaveSettings(ctx, candidate)
		require.Error(t, err, invalid)
		require.Equal(t, original, repo.values[identitySettingsSetting])
	}
	for _, limits := range [][2]int{{0, 4}, {3, 4}, {1, 0}, {1, 11}, {-1, 4}} {
		candidate := *settings
		candidate.ConcurrentAccounts, candidate.ProbeConcurrency = limits[0], limits[1]
		_, err := svc.SaveSettings(ctx, candidate)
		require.Error(t, err)
		require.Equal(t, original, repo.values[identitySettingsSetting], "invalid limits must not partially replace settings")
	}
	repo.writeErr = errors.New("storage unavailable")
	_, err = svc.SaveSettings(ctx, IdentitySettings{PublicBaseURL: "https://changed.example.com", ConcurrentAccounts: 1, ProbeConcurrency: 1})
	require.Error(t, err)
	require.Equal(t, original, repo.values[identitySettingsSetting])
}

func TestIdentitySettingsLegacyBaseURLAndStorageFailures(t *testing.T) {
	repo := &identitySettingsRepo{values: map[string]string{identityPublicURLSetting: "https://existing.example.com"}}
	svc := NewModelIdentityService(nil, nil, nil, repo)
	ctx := context.Background()
	settings, err := svc.Settings(ctx)
	require.NoError(t, err)
	require.Equal(t, "https://existing.example.com", settings.PublicBaseURL)
	require.Equal(t, 1, settings.ConcurrentAccounts)
	require.Equal(t, 4, settings.ProbeConcurrency)
	for _, invalid := range []string{"not-json", `{"concurrent_accounts":0,"probe_concurrency":4}`, `{"concurrent_accounts":1,"probe_concurrency":11}`} {
		repo.values[identitySettingsSetting] = invalid
		_, err = svc.Settings(ctx)
		require.Error(t, err, "must not silently fall back after a stored settings error")
	}
	repo.readErr = errors.New("database unavailable")
	_, err = svc.Settings(ctx)
	require.ErrorContains(t, err, "database unavailable")
}
