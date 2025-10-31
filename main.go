package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

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
	if v, err := g.SetView("list", 0, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Go-Dash - Your TUI Dashboard"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		fmt.Fprintln(v, "Welcome to Go-Dash! Select an application and press Enter to launch.")
		fmt.Fprintln(v, "")

		for _, app := range applications {
			fmt.Fprintln(v, app.Name)
		}

		if _, err := g.SetCurrentView("list"); err != nil {
			return err
		}

		if err := v.SetCursor(0, 2); err != nil {
			return err
		}
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
	return nil
}

// cursorUp moves the cursor up in the list
func cursorUp(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	if y > 2 {
		v.MoveCursor(0, -1, false)
	}
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

// runApp executes the selected application
func runApp(g *gocui.Gui, v *gocui.View) error {
	app := getAppFromCursor(v)
	if app == nil {
		return nil
	}

	// we need to stop gocui and then run the command
	// after the command is finished, we need to restart gocui
	// to do that, we will return a special error

	cmd := exec.Command("sh", "-c", app.Command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Println("Error running command:", err)
	}

	// By returning ErrRestart, we are signaling the main loop to restart the application
	return ErrRestart
}
