package ui

import (
	"testing"

	"github.com/jesspatton/lazytest/engine"
)

func TestNotification_Lifecycle(t *testing.T) {
	eng := engine.New(t.TempDir())
	m := NewModel(eng)

	// Initial state
	if m.activeNotification != "" {
		t.Errorf("Expected empty active notification initially, got %q", m.activeNotification)
	}

	// Send NotificationMsg
	updatedModel, cmd := m.Update(engine.NotificationMsg{
		Message: "Test Error",
		IsError: true,
	})
	m = updatedModel.(Model)

	if m.activeNotification != "Test Error" {
		t.Errorf("Expected active notification 'Test Error', got %q", m.activeNotification)
	}
	if !m.isNotificationError {
		t.Error("Expected isNotificationError to be true")
	}
	if cmd == nil {
		t.Error("Expected a tick command for clearing notification")
	}

	// Verify footer view contains notification text
	footer := m.renderFooter()
	if len(footer) == 0 {
		t.Error("Expected non-empty footer string")
	}

	// Send ClearNotificationMsg matching ID
	updatedModel, _ = m.Update(engine.ClearNotificationMsg{ID: m.notificationID})
	m = updatedModel.(Model)

	if m.activeNotification != "" {
		t.Errorf("Expected active notification to be cleared, got %q", m.activeNotification)
	}
	if m.isNotificationError {
		t.Error("Expected isNotificationError to be false after clear")
	}
}

func TestNotification_MismatchedClearID(t *testing.T) {
	eng := engine.New(t.TempDir())
	m := NewModel(eng)

	m.activeNotification = "Newer Notification"
	m.isNotificationError = true
	m.notificationID = 2

	// Send ClearNotificationMsg with older ID (1)
	updatedModel, _ := m.Update(engine.ClearNotificationMsg{ID: 1})
	m = updatedModel.(Model)

	if m.activeNotification != "Newer Notification" {
		t.Errorf("Notification should not be cleared by outdated ID, got %q", m.activeNotification)
	}
}
