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

	"secretnotes-cli/internal/api"
)

type EditorApp struct {
	client      *api.Client
	pass        []byte
	serverName  string

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

	// data
	loaded      bool
	initialErr  error
}

func NewEditorApp(client *api.Client, passphrase []byte, serverName string, autosave bool, debounce time.Duration) *EditorApp {
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
	}
}

// Run starts the Bubble Tea program
func (a *EditorApp) Run(ctx context.Context) error {
	p := tea.NewProgram(a, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

// Init loads note
func (a *EditorApp) Init() tea.Cmd {
	return a.loadNoteCmd()
}

func (a *EditorApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		s := m.String()
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
		case "ctrl+c", "ctrl+q":
			return a, tea.Quit
		}
	case loadedMsg:
		if m.err != nil {
			a.initialErr = m.err
			a.ta.Placeholder = "Failed to load note (press Ctrl+Q to quit)"
			a.status = fmt.Sprintf("Error: %v", m.err)
			return a, nil
		}
		a.loaded = true
		a.ta.SetValue(m.note.Message)
		a.ta.Placeholder = "Start typing your secure note..."
		a.status = "Loaded"
		return a, nil
case savedMsg:
		if m.err != nil {
			a.status = fmt.Sprintf("Save failed: %v", m.err)
			return a, nil
		}
		a.lastSaved = time.Now()
		a.status = fmt.Sprintf("Saved at %s", a.lastSaved.Format("15:04:05"))
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
	status := fmt.Sprintf("Server: %s  |  Autosave: %v  |  %s", a.serverName, a.autosave, a.status)
	base := border.Render(a.ta.View()) + "\n" + lipgloss.NewStyle().Faint(true).Render(status)
	if a.prompting {
		modalBorder := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(1, 2)
		title := lipgloss.NewStyle().Bold(true).Render("Change passphrase")
		prompt := "New passphrase: " + a.pin.View() + "\nPress Enter to load, Esc to cancel"
		modal := modalBorder.Render(title+"\n"+prompt)
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