package ui

import (
	"fmt"
	"os"
	"syscall"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type progressModel struct {
	bar     progress.Model
	label   string
	current int
	total   int
	last    string
	done    bool
}

type progressTrackMsg struct{ name string }
type progressDoneMsg struct{}

func (m progressModel) Init() tea.Cmd { return nil }

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressTrackMsg:
		m.current++
		m.last = msg.name
		pct := float64(m.current) / float64(m.total)
		cmd := m.bar.SetPercent(pct)
		return m, cmd
	case progressDoneMsg:
		m.done = true
		return m, tea.Quit
	case progress.FrameMsg:
		updated, cmd := m.bar.Update(msg)
		m.bar = updated.(progress.Model)
		return m, cmd
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			syscall.Kill(0, syscall.SIGINT)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m progressModel) View() string {
	if m.done {
		return ""
	}
	counter := styleDim.Render(fmt.Sprintf("%d / %d", m.current, m.total))
	last := ""
	if m.last != "" {
		last = "\n  " + styleSuccess.Render("✓") + "  " + styleDim.Render(m.last)
	}
	return fmt.Sprintf("\n  %s\n  %s  %s%s\n",
		styleStep.Render(m.label),
		m.bar.View(),
		counter,
		last,
	)
}

// TrackProgress shows a Bubble Tea progress bar for multi-track operations.
// Subprocess output is silenced at the source (ytdlp.Run, etc.) so no stdout
// redirect is needed here — Bubble Tea writes directly to os.Stdout cleanly.
type TrackProgress struct {
	program *tea.Program
	doneCh  chan struct{}
}

// NewTrackProgress starts a progress bar for total tracks under label.
// Returns nil in verbose mode — callers must nil-check before calling Track/Done.
func NewTrackProgress(label string, total int) *TrackProgress {
	if Verbose {
		Step(fmt.Sprintf("%s  (%d tracks)", label, total))
		return nil
	}

	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(36),
		progress.WithoutPercentage(),
	)

	m := progressModel{bar: bar, label: label, total: total}
	p := tea.NewProgram(m, tea.WithOutput(os.Stdout))
	tp := &TrackProgress{
		program: p,
		doneCh:  make(chan struct{}),
	}

	go func() {
		p.Run()
		close(tp.doneCh)
	}()

	return tp
}

// Track marks one track completed and advances the bar.
func (tp *TrackProgress) Track(name string) {
	if tp == nil {
		Success(name)
		return
	}
	tp.program.Send(progressTrackMsg{name: name})
}

// Done stops the bar and waits for it to finish rendering.
func (tp *TrackProgress) Done() {
	if tp == nil {
		return
	}
	tp.program.Send(progressDoneMsg{})
	<-tp.doneCh
}
