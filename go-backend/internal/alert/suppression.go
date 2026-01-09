package alert

import (
	"sync"
	"time"
)

// SuppressionEntry represents a suppressed alert
type SuppressionEntry struct {
	Key       string
	ExpiresAt time.Time
}

// SuppressionManager manages alert suppression to prevent alert flooding
type SuppressionManager struct {
	suppressions  map[string]time.Time
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCh        chan struct{}
}

// NewSuppressionManager creates a new suppression manager
func NewSuppressionManager() *SuppressionManager {
	sm := &SuppressionManager{
		suppressions: make(map[string]time.Time),
		stopCh:       make(chan struct{}),
	}

	// Start cleanup ticker (every minute)
	sm.cleanupTicker = time.NewTicker(time.Minute)
	go sm.cleanupLoop()

	return sm
}

// AddSuppression adds a suppression entry
func (sm *SuppressionManager) AddSuppression(key string, duration time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.suppressions[key] = time.Now().Add(duration)
}

// IsSuppressed checks if an alert is currently suppressed
func (sm *SuppressionManager) IsSuppressed(key string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	expiresAt, exists := sm.suppressions[key]
	if !exists {
		return false
	}

	if time.Now().After(expiresAt) {
		// Expired, remove it
		delete(sm.suppressions, key)
		return false
	}

	return true
}

// RemoveSuppression removes a suppression entry
func (sm *SuppressionManager) RemoveSuppression(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.suppressions, key)
}

// GetActiveSuppressions returns all active suppressions
func (sm *SuppressionManager) GetActiveSuppressions() []SuppressionEntry {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var entries []SuppressionEntry
	now := time.Now()

	for key, expiresAt := range sm.suppressions {
		if now.Before(expiresAt) {
			entries = append(entries, SuppressionEntry{
				Key:       key,
				ExpiresAt: expiresAt,
			})
		}
	}

	return entries
}

// Stop stops the suppression manager
func (sm *SuppressionManager) Stop() {
	close(sm.stopCh)
	if sm.cleanupTicker != nil {
		sm.cleanupTicker.Stop()
	}
}

// cleanupLoop periodically removes expired suppressions
func (sm *SuppressionManager) cleanupLoop() {
	for {
		select {
		case <-sm.cleanupTicker.C:
			sm.cleanup()
		case <-sm.stopCh:
			return
		}
	}
}

// cleanup removes expired suppressions
func (sm *SuppressionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for key, expiresAt := range sm.suppressions {
		if now.After(expiresAt) {
			delete(sm.suppressions, key)
		}
	}
}
