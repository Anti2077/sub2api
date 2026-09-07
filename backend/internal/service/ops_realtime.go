package service

import (
	"context"
	"errors"
	"strings"
)

// IsRealtimeMonitoringEnabled returns true when realtime ops features are enabled.
//
// This is a soft switch controlled by the DB setting `ops_realtime_monitoring_enabled`,
// and it is also gated by the hard switch/soft switch of overall ops monitoring.
func (s *OpsService) IsRealtimeMonitoringEnabled(ctx context.Context) bool {
	if !s.IsMonitoringEnabled(ctx) {
		return false
	}
	if snapshot := s.runtimeSettings.Load(); snapshot != nil {
		return snapshot.realtimeEnabled
	}
	if s.settingRepo == nil {
		return true
	}

	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpsRealtimeMonitoringEnabled)
	if err != nil {
		// Default enabled when key is missing; fail-open on transient errors.
		if errors.Is(err, ErrSettingNotFound) {
			return true
		}
		return true
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "off", "disabled":
		return false
	default:
		return true
	}
}

// SetRealtimeMonitoringEnabled updates the hot-path snapshot after an admin
// settings write, avoiding a Redis/DB lookup for every gateway request.
func (s *OpsService) SetRealtimeMonitoringEnabled(enabled bool) {
	if s == nil {
		return
	}
	s.runtimeSettingsMu.Lock()
	current := s.runtimeSettings.Load()
	next := &opsRuntimeSettingsSnapshot{monitoringEnabled: true, realtimeEnabled: enabled, advanced: *defaultOpsAdvancedSettings()}
	if current != nil {
		next.monitoringEnabled = current.monitoringEnabled
		next.advanced = current.advanced
	}
	s.runtimeSettings.Store(next)
	s.runtimeSettingsMu.Unlock()
}
