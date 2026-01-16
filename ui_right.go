package main

// RenderRightPaneWrapper is a thin wrapper around the internal rendering helper
// to expose a dedicated Right Pane renderer.
func RenderRightPaneWrapper(title, body string, width int) []string {
	return renderRightPane(title, body, width)
}
