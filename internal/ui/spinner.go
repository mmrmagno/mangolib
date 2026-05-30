package ui

import (
	"os"
	"syscall"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type spinnerModel struct {
	spinner spinner.Model
	label   string
	done    bool
}

type spinnerDoneMsg struct{ err error }

func (m spinnerModel) Init() tea.Cmd { return m.spinner.Tick }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDoneMsg:
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			// Forward as a real signal so subprocesses (yt-dlp, rsync) also terminate.
			syscall.Kill(0, syscall.SIGINT)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return m.spinner.View() + "  " + styleDim.Render(m.label) + "\n"
}

// RunWithSpinner runs fn while showing an animated spinner.
// Subprocess output is routed to io.Discard (see ytdlp.Run, ipod.runSync) so
// there is no conflict with Bubble Tea rendering — no stdout redirect needed here.
// In verbose mode the spinner is skipped and fn runs with full output.
func RunWithSpinner(label string, fn func() error) error {
	if Verbose {
		Step(label)
		return fn()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMauve))

	m := spinnerModel{spinner: s, label: label}
	p := tea.NewProgram(m, tea.WithOutput(os.Stdout))

	errCh := make(chan error, 1)
	go func() {
		err := fn()
		errCh <- err
		p.Send(spinnerDoneMsg{err: err})
	}()

	p.Run()

	result := <-errCh
	return result
}
