package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ktappdev/secretnotes-go-backend/cli/internal/api"
)

type EditorApp struct {
	client      *api.Client
	pass        []byte
	serverName  string

	// exit semantics
	exitMode    string // "wipe" clears screen+scrollback; "clear" clears screen only

	// UI state
	ta          textarea.Model
	status      string
	lastSaved   time.Time
	autosave    bool
	debounce    time.Duration
	saveTimer   *time.Timer
	seq         int // debounce sequence

	// Passphrase prompt
	prompting   bool
	pin         textinput.Model

	// View modes
	plainCopyMode bool
	showAbout     bool

	// connectivity
	connected  bool

	// persistence
	savePref    func(enabled bool, debounceMs int) error

	// data
	loaded      bool
	initialErr  error
}

func NewEditorApp(client *api.Client, passphrase []byte, serverName string, autosave bool, debounce time.Duration, savePref func(bool, int) error) *EditorApp {
	ta := textarea.New()
	ta.Placeholder = "Loading note..."
	ta.Focus()
	// reasonable sizes; Bubble Tea will reflow in terminal
	ta.SetWidth(100)
	ta.SetHeight(24)

	pin := textinput.New()
	pin.Placeholder = "Enter new passphrase"
	pin.Prompt = ""
	pin.EchoMode = textinput.EchoPassword
	pin.EchoCharacter = '•'
	pin.CharLimit = 256
	pin.Width = 48

	return &EditorApp{
		client:     client,
		pass:       passphrase,
		serverName: serverName,
		ta:         ta,
		status:     "",
		autosave:   autosave,
		debounce:   debounce,
		pin:        pin,
		savePref:   savePref,
	}
}

// Run starts the Bubble Tea program
func (a *EditorApp) Run(ctx context.Context) error {
	p := tea.NewProgram(a, tea.WithContext(ctx), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ExitMode reports how the user exited the app: "wipe" or "clear".
func (a *EditorApp) ExitMode() string { return a.exitMode }

// Init loads note
func (a *EditorApp) Init() tea.Cmd {
	return a.loadNoteCmd()
}

func (a *EditorApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		s := m.String()
		// About overlay interaction
		if a.showAbout {
			switch s {
			case "?", "esc", "enter", "ctrl+c":
				a.showAbout = false
				return a, nil
			}
			return a, nil
		}
		if a.prompting {
			// Passphrase prompt interactions
			switch s {
			case "enter":
				val := a.pin.Value()
				if len(val) < 3 {
					a.status = "Passphrase must be at least 3 characters"
					return a, nil
				}
				// swap passphrase (best-effort zero existing buffer)
				for i := range a.pass { a.pass[i] = 0 }
				a.pass = []byte(val)
				a.prompting = false
				a.pin.Reset()
				a.ta.Focus()
				return a, a.loadNoteCmd()
			case "esc", "ctrl+c":
				a.prompting = false
				a.pin.Reset()
				a.ta.Focus()
				return a, nil
			default:
				var cmd tea.Cmd
				a.pin, cmd = a.pin.Update(m)
				return a, cmd
			}
		}
		switch s {
		case "?":
			a.showAbout = !a.showAbout
			return a, nil
		case "ctrl+shift+s", "ctrl+S", "alt+s":
			// Toggle autosave and persist preference
			a.autosave = !a.autosave
			if a.savePref != nil {
				_ = a.savePref(a.autosave, int(a.debounce/time.Millisecond))
			}
			if a.autosave { a.status = "Autosave: on" } else { a.status = "Autosave: off" }
			return a, nil
		case "ctrl+s":
			return a, a.saveCmd()
		case "ctrl+p":
			a.prompting = true
			a.pin.SetValue("")
			a.pin.Focus()
			a.ta.Blur()
			a.status = "Enter new passphrase and press Enter"
			return a, nil
case "ctrl+t":
			a.plainCopyMode = !a.plainCopyMode
			if a.plainCopyMode {
				a.status = "Plain view: only your text is shown for easy copying"
			} else {
				a.status = "Normal view"
			}
			return a, nil
		case "ctrl+y":
			// Copy raw content to clipboard
			_ = clipboard.WriteAll(a.ta.Value())
			a.status = "Copied note to clipboard"
			return a, nil
		case "ctrl+q":
			a.exitMode = "wipe"
			return a, tea.Quit
		case "ctrl+c":
			a.exitMode = "clear"
			return a, tea.Quit
		}
case loadedMsg:
		if m.err != nil {
			a.initialErr = m.err
			a.connected = false
			a.ta.Placeholder = "Failed to load note (press Ctrl+Q to quit)"
			a.status = "Offline"
			return a, nil
		}
		a.loaded = true
		a.connected = true
	a.ta.SetValue(m.note.Message)
		a.ta.Placeholder = "Start typing your secure note..."
		// Clear transient status to avoid duplicate "Connected" in footer
		a.status = ""
		return a, nil
case savedMsg:
		if m.err != nil {
			a.connected = false
			a.status = "Offline (save failed)"
			return a, nil
		}
		a.connected = true
		a.lastSaved = time.Now()
		a.status = fmt.Sprintf("Saved %s", a.lastSaved.Format("15:04:05"))
		return a, nil
	case autoSaveMsg:
		// Only save if token matches the latest sequence
		if m.seq == a.seq {
			return a, a.saveCmd()
		}
		return a, nil
	}

	// Delegate to textarea
	var cmd tea.Cmd
	prev := a.ta.Value()
	a.ta, cmd = a.ta.Update(msg)
	// If content changed and autosave enabled, schedule debounced save via sequence token
	if a.autosave && a.loaded && a.ta.Value() != prev {
		a.seq++
		seq := a.seq
		deb := a.debounce
		return a, tea.Tick(deb, func(time.Time) tea.Msg { return autoSaveMsg{seq: seq} })
	}
	return a, cmd
}

func (a *EditorApp) View() string {
	if a.plainCopyMode {
		// Show only the raw note content with no UI, borders, or line numbers.
		return a.ta.Value()
	}
	border := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Padding(0, 1)
	conn := "Offline"
	if a.connected {
		conn = "Connected"
	}
	status := fmt.Sprintf("Status: %s  |  Autosave: %v", conn, a.autosave)
	if a.status != "" && a.status != "Connected" {
		status = fmt.Sprintf("%s  |  %s", status, a.status)
	}
	base := border.Render(a.ta.View()) + "\n" + lipgloss.NewStyle().Faint(true).Render(status)
	// footer hints
	hints := "?: About • Ctrl+T Plain • Ctrl+Y Copy • Ctrl+P Passphrase • Alt+S Autosave • Ctrl+S Save • Ctrl+Q Quit"
	base = base + "\n" + lipgloss.NewStyle().Faint(true).Render(hints)
	if a.showAbout {
		// About modal
		header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Render("SecretNotes CLI")
		sub := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("135")).Render("Created by Clint and Ken for Lugetech")
		body := "A zero‑knowledge, passphrase‑based, single‑note editor.\n" +
			"One passphrase → one note, encrypted end‑to‑end.\n" +
			"No accounts, no tracking — your secret stays yours.\n" +
			"Text‑first TUI with save, autosave, and quick copy.\n" +
			"Privacy: Ctrl+Q wipes screen + history, Ctrl+C clears screen only.\n" +
			"Note: Not all terminals clear scrollback; for maximum privacy close your terminal or run with SN_WIPE_AGGRESSIVE=1."
		warn := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("Important: If you forget your passphrase, your note is permanently unrecoverable.")
		modalBorder := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(1, 2)
		modal := modalBorder.Render(header+"\n"+sub+"\n\n"+body+"\n\n"+warn+"\n\nPress ? or Esc to close")
		return base + "\n" + modal
	}
	if a.prompting {
		modalBorder := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(1, 2)
		title := lipgloss.NewStyle().Bold(true).Render("Change passphrase")
		prompt := "New passphrase: " + a.pin.View() + "\nPress Enter to load, Esc to cancel"
		warn := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("No recovery: If you forget your passphrase, nobody can recover your note.")
		modal := modalBorder.Render(title+"\n"+prompt+"\n\n"+warn)
		return base + "\n" + modal
	}
	return base
}

// Messages and commands

type loadedMsg struct{ note *api.Note; err error }
type savedMsg struct{ note *api.Note; err error }
type autoSaveMsg struct{ seq int }

func (a *EditorApp) loadNoteCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		note, err := a.client.GetOrCreateNote(ctx, a.pass)
		return loadedMsg{note: note, err: err}
	}
}

func (a *EditorApp) saveCmd() tea.Cmd {
	content := a.ta.Value()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		note, err := a.client.UpdateNote(ctx, a.pass, content)
		return savedMsg{note: note, err: err}
	}
}