package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/jroimartin/gocui"
)

// Config holds the list of applications
type Config struct {
	Applications []Application `json:"applications"`
}

// Application represents a TUI application
// with a name and the command to execute it
type Application struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

var applications []Application
var ErrRestart = errors.New("restart")
var pendingApp *Application

func main() {
	// Create a default config file if it does not exist
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		createDefaultConfig()
	}

	// Load the configuration
	config, err := loadConfig("config.json")
	if err != nil {
		log.Panicln("Error loading config:", err)
	}
	applications = config.Applications

	for {
		if err := run(); err != nil {
			if err == ErrRestart {
				// Check if there's a pending app to run
				if pendingApp != nil {
					app := pendingApp
					pendingApp = nil
					runApplication(app)
				}
				continue
			}
			if err != gocui.ErrQuit {
				log.Panicln(err)
			}
			break
		}
	}
}

func run() error {
	// Initialize gocui
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return err
	}
	defer g.Close()

	// Show the text cursor for editable views
	g.Cursor = true

	g.SetManagerFunc(layout)

	// Keybindings
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("list", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("list", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("list", gocui.KeyEnter, gocui.ModNone, runApp); err != nil {
		return err
	}
	if err := g.SetKeybinding("list", 'r', gocui.ModNone, refreshDashboard); err != nil {
		return err
	}
	// Add new application (only when focused on list)
	if err := g.SetKeybinding("list", 'a', gocui.ModNone, openAddModal); err != nil {
		return err
	}
	// Modal controls
	if err := g.SetKeybinding("add_name", gocui.KeyEnter, gocui.ModNone, switchAddField); err != nil {
		// ignore at startup; views may not exist yet
	}
	if err := g.SetKeybinding("add_cmd", gocui.KeyEnter, gocui.ModNone, saveNewApp); err != nil {
	}
	if err := g.SetKeybinding("add_name", gocui.KeyTab, gocui.ModNone, switchAddField); err != nil {
	}
	if err := g.SetKeybinding("add_cmd", gocui.KeyTab, gocui.ModNone, switchAddField); err != nil {
	}
	if err := g.SetKeybinding("add_modal", gocui.KeyEsc, gocui.ModNone, cancelAddModal); err != nil {
	}

	// Main loop
	return g.MainLoop()
}

// loadConfig reads and parses the config.json file
func loadConfig(path string) (Config, error) {
	var config Config
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(file, &config)
	return config, err
}

// createDefaultConfig creates a default config.json file
func createDefaultConfig() {
	config := Config{
		Applications: []Application{
			{Name: "LazyGit", Command: "lazygit"},
			{Name: "LazyDocker", Command: "lazydocker"},
			{Name: "LazySSH", Command: "lazyssh"},
		},
	}

	file, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Panicln("Error creating default config:", err)
	}

	if err := ioutil.WriteFile("config.json", file, 0644); err != nil {
		log.Panicln("Error creating default config:", err)
	}
}

// layout sets up the view
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// Determine sidebar width (about 30% of the screen, min 24 cols)
	sidebarWidth := maxX / 3
	if sidebarWidth < 24 {
		sidebarWidth = 24
	}
	if sidebarWidth > maxX-30 { // keep space for dashboard
		sidebarWidth = maxX - 30
		if sidebarWidth < 24 {
			sidebarWidth = 24
		}
	}

	// Sidebar: application list
	if v, err := g.SetView("list", 0, 0, sidebarWidth-1, maxY-2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Go-Dash • Links"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		renderList(g)

		if _, err := g.SetCurrentView("list"); err != nil {
			return err
		}
		if err := v.SetCursor(0, 2); err != nil {
			return err
		}
	}

	// Dashboard: right panel
	if dv, err := g.SetView("dashboard", sidebarWidth, 0, maxX-1, maxY-2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		dv.Title = "New Tab • Dashboard"
		renderDashboard(g)
	}

	// Footer/status bar
	if fv, err := g.SetView("footer", 0, maxY-2, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fv.Frame = false
		fmt.Fprintf(fv, "  ↑/↓ Move   Enter Launch   a Add   r Refresh   Ctrl+C Quit  ")
	}

	return nil
}

// quit is the handler for Ctrl+C
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// cursorDown moves the cursor down in the list
func cursorDown(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	if y < len(applications)+1 {
		v.MoveCursor(0, 1, false)
	}
	// Refresh dashboard to reflect selection change
	renderDashboard(g)
	return nil
}

// cursorUp moves the cursor up in the list
func cursorUp(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	if y > 2 {
		v.MoveCursor(0, -1, false)
	}
	// Refresh dashboard to reflect selection change
	renderDashboard(g)
	return nil
}

// getAppFromCursor returns the application from the current cursor position
func getAppFromCursor(v *gocui.View) *Application {
	_, y := v.Cursor()
	// The first two lines are the title and a blank line
	index := y - 2
	if index >= 0 && index < len(applications) {
		return &applications[index]
	}
	return nil
}

// runApp stores the app to run and triggers gocui shutdown
func runApp(g *gocui.Gui, v *gocui.View) error {
	app := getAppFromCursor(v)
	if app == nil {
		return nil
	}

	// Store the app to run after gocui closes
	pendingApp = app

	// Return ErrRestart to close gocui, which will restore the terminal
	// The main loop will then call runApplication to actually run the command
	return ErrRestart
}

// runApplication executes the application after gocui has closed
func runApplication(app *Application) {
	cmd := exec.Command("sh", "-c", app.Command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Println("Error starting command:", err)
		return
	}

	// Set up signal forwarding to the child process
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Forward signals to the child process
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	// Wait for command to finish
	err := <-done

	// Stop forwarding signals
	signal.Stop(sigChan)
	close(sigChan)

	if err != nil {
		// Ignore exit status from signal interrupts (Ctrl+C) as normal exits
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				if status.Signal() == syscall.SIGINT || status.Signal() == syscall.SIGTERM {
					// Normal exit via Ctrl+C
					return
				}
			}
		}
		// Only log if it's not a normal signal-based exit
		log.Println("Error running command:", err)
	}
}

// renderDashboard paints the right-side panel with a "new tab" style
func renderDashboard(g *gocui.Gui) {
	g.Update(func(gg *gocui.Gui) error {
		dv, err := gg.View("dashboard")
		if err != nil {
			return nil
		}
		dv.Clear()

		// Gather dynamic info
		hostname, _ := os.Hostname()
		cwd, _ := os.Getwd()
		now := time.Now()

		// Selected app info (if any)
		var selectedName string
		var selectedCmd string
		if lv, lerr := gg.View("list"); lerr == nil {
			if app := getAppFromCursor(lv); app != nil {
				selectedName = app.Name
				selectedCmd = app.Command
			}
		}

		// Header
		fmt.Fprintf(dv, "Time: %s\n", now.Format("Mon Jan 2, 2006 15:04:05"))
		fmt.Fprintf(dv, "Host: %s\n", hostname)
		fmt.Fprintf(dv, "Dir:  %s\n", cwd)
		fmt.Fprintln(dv, "")

		// Featured tile
		fmt.Fprintln(dv, "── Featured ─────────────────────────────────────────")
		fmt.Fprintln(dv, "Pro tip: Bookmark your favorite TUI tools in config.json.")
		fmt.Fprintln(dv, "• Keep sessions fast. • Launch with Enter. • Quit with Ctrl+C.")
		fmt.Fprintln(dv, "")

		// Selected app details
		fmt.Fprintln(dv, "── Selection ────────────────────────────────────────")
		if selectedName != "" {
			fmt.Fprintf(dv, "App: %s\n", selectedName)
			fmt.Fprintf(dv, "Cmd: %s\n", selectedCmd)
		} else {
			fmt.Fprintln(dv, "No app selected. Use ↑/↓ to choose from the left.")
		}
		fmt.Fprintln(dv, "")

		// Quick actions
		fmt.Fprintln(dv, "── Quick Actions ────────────────────────────────────")
		fmt.Fprintln(dv, "[Enter] Launch selection   [r] Refresh dashboard")
		fmt.Fprintln(dv, "")

		// ASCII brand
		fmt.Fprintln(dv, "── Go-Dash ──────────────────────────────────────────")
		fmt.Fprintln(dv, "   _____       ____           _     ")
		fmt.Fprintln(dv, "  / ____|     |  _ \\\\         | |    ")
		fmt.Fprintln(dv, " | |  __  ___ | |_) | __ _ ___| |__  ")
		fmt.Fprintln(dv, " | | |_ |/ _ \\\\|  _ < / _` / __| '_ \\")
		fmt.Fprintln(dv, " | |__| | (_) | |_) | (_| \\\\__ \\\\ | | |")
		fmt.Fprintln(dv, "  \\\\_____|\\\\___/|____/ \\\\__,_|___/_| |_|")
		return nil
	})
}

// refreshDashboard is a keybinding handler to manually refresh the dashboard
func refreshDashboard(g *gocui.Gui, v *gocui.View) error {
	renderDashboard(g)
	return nil
}

// renderList refreshes the left sidebar list content
func renderList(g *gocui.Gui) {
	g.Update(func(gg *gocui.Gui) error {
		lv, err := gg.View("list")
		if err != nil {
			return nil
		}
		lv.Clear()
		fmt.Fprintln(lv, "Welcome to Go-Dash! Select an app and press Enter.")
		fmt.Fprintln(lv, "")
		for _, app := range applications {
			fmt.Fprintln(lv, app.Name)
		}
		return nil
	})
}

// openAddModal shows a centered modal with two inputs: name and command
func openAddModal(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	width := 64
	height := 12
	if width > maxX-4 {
		width = maxX - 4
	}
	if height > maxY-4 {
		height = maxY - 4
	}
	left := (maxX - width) / 2
	top := (maxY - height) / 2
	right := left + width
	bottom := top + height

	if mv, err := g.SetView("add_modal", left, top, right, bottom); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		mv.Title = "Add Application"
		mv.Wrap = false
		fmt.Fprintln(mv, "Enter details below.")
		fmt.Fprintln(mv, "Press Enter on Command to save. Esc cancels.")
	}
	// Name field (height 3 => 1 inner text line with frame)
	if nv, err := g.SetView("add_name", left+2, top+3, right-2, top+5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		nv.Title = "Name"
		nv.Editable = true
		nv.Editor = gocui.DefaultEditor
	}
	// Command field (height 3 => 1 inner text line with frame)
	if cv, err := g.SetView("add_cmd", left+2, top+8, right-2, top+10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		cv.Title = "Command"
		cv.Editable = true
		cv.Editor = gocui.DefaultEditor
	}

	if _, err := g.SetCurrentView("add_name"); err != nil {
		return err
	}
	return nil
}

// switchAddField moves focus between name and command inputs
func switchAddField(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}
	if v.Name() == "add_name" {
		_, err := g.SetCurrentView("add_cmd")
		return err
	}
	_, err := g.SetCurrentView("add_name")
	return err
}

// saveNewApp reads fields, appends to config, writes file, refreshes UI
func saveNewApp(g *gocui.Gui, v *gocui.View) error {
	nameV, err := g.View("add_name")
	if err != nil {
		return err
	}
	cmdV, err := g.View("add_cmd")
	if err != nil {
		return err
	}
	name := trimViewText(nameV)
	cmd := trimViewText(cmdV)
	if name == "" || cmd == "" {
		return nil
	}
	applications = append(applications, Application{Name: name, Command: cmd})
	if err := writeConfig("config.json", applications); err != nil {
		log.Println("Error writing config:", err)
	}
	// close modal
	cancelAddModal(g, nil)
	// refresh
	renderList(g)
	renderDashboard(g)
	return nil
}

// cancelAddModal removes modal and input views
func cancelAddModal(g *gocui.Gui, v *gocui.View) error {
	for _, name := range []string{"add_modal", "add_name", "add_cmd"} {
		if cv, err := g.View(name); err == nil {
			g.DeleteView(cv.Name())
		}
	}
	// Return focus to list
	if _, err := g.SetCurrentView("list"); err != nil {
		return err
	}
	return nil
}

// trimViewText returns the content of a view's buffer as a single trimmed line
func trimViewText(v *gocui.View) string {
	text := v.Buffer()
	// remove trailing newlines that gocui keeps in buffer
	for len(text) > 0 {
		last := text[len(text)-1]
		if last == '\n' || last == '\r' {
			text = text[:len(text)-1]
			continue
		}
		break
	}
	return text
}

// writeConfig persists the applications to disk
func writeConfig(path string, apps []Application) error {
	conf := Config{Applications: apps}
	file, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, file, 0644)
}
