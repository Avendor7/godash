package main

// RenderLeftPanelBlock is a thin wrapper around the internal rendering helper
// to expose a dedicated UI module for left panels.
func RenderLeftPanelBlock(title string, items []Item, cursor int, focused bool, width int) []string {
	return renderPanelBlock(title, items, cursor, focused, width)
}
