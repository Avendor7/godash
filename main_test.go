package main

import "testing"

func TestInitialActivationTop(t *testing.T) {
	m := initialModel()
	m.activateCurrent()
	if m.rightTitle != "TUI Apps: Clock" || m.rightBody != "A terminal clock widget" {
		t.Fatalf("unexpected right pane: title=%q body=%q", m.rightTitle, m.rightBody)
	}
}

func TestCycleAndActivateDirs(t *testing.T) {
	m := initialModel()
	m.cycleFocus() // move to PanelDirs
	m.activateCurrent()
	if m.rightTitle != "Directory: Dev Projects" || m.rightBody != "Path: /home/you/projects" {
		t.Fatalf("unexpected right pane after activating first dir: title=%q body=%q", m.rightTitle, m.rightBody)
	}
}

func TestNavigateDownTopPanel(t *testing.T) {
	m := initialModel()
	m.navigateDown()
	if m.topCursor != 1 {
		t.Fatalf("expected topCursor to be 1 after navigateDown, got %d", m.topCursor)
	}
}
