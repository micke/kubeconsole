package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	apiv1 "k8s.io/api/core/v1"
)

// Message types sent from external goroutines via p.Send()
type PodEventMsg struct{ Message string }
type PodReadyMsg struct{ Pod *apiv1.Pod }
type PodErrorMsg struct{ Message string }

// StartupResult holds the outcome of the spinner UI.
type StartupResult struct {
	Pod       *apiv1.Pod
	Cancelled bool
	Error     error
}

type phase int

const (
	phaseStarting  phase = iota
	phaseAttaching
	phaseDone
	phaseError
)

type attachTickMsg struct{}

type spinnerModel struct {
	spinner       spinner.Model
	phase         phase
	podName       string
	containerName string
	isNewPod      bool
	currentEvent  string
	readyPod      *apiv1.Pod
	cancelled     bool
	err           error
}

var (
	GreenCheck = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✓")
	RedCross   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗")
)

func newSpinnerModel(podName, containerName string, isNewPod bool) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	startPhase := phaseStarting
	if !isNewPod {
		startPhase = phaseAttaching
	}

	return spinnerModel{
		spinner:       s,
		phase:         startPhase,
		podName:       podName,
		containerName: containerName,
		isNewPod:      isNewPod,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.cancelled = true
			return m, tea.Quit
		}

	case PodEventMsg:
		if m.phase == phaseStarting {
			m.currentEvent = msg.Message
		}
		return m, nil

	case PodReadyMsg:
		m.readyPod = msg.Pod
		if m.phase == phaseStarting {
			m.phase = phaseAttaching
			m.currentEvent = ""
			return m, tea.Sequence(
				tea.Println(GreenCheck+" Pod ready"),
				func() tea.Msg {
					time.Sleep(500 * time.Millisecond)
					return attachTickMsg{}
				},
			)
		}
		// If already in attaching phase (re-attach case), complete
		if m.phase == phaseAttaching {
			m.phase = phaseDone
			return m, tea.Sequence(
				tea.Println(GreenCheck+" Attached to "+m.containerName),
				tea.Quit,
			)
		}
		return m, nil

	case attachTickMsg:
		m.phase = phaseDone
		return m, tea.Sequence(
			tea.Println(GreenCheck+" Attached to "+m.containerName),
			tea.Quit,
		)

	case PodErrorMsg:
		m.phase = phaseError
		m.err = fmt.Errorf("%s", msg.Message)
		return m, tea.Sequence(
			tea.Println(RedCross+" "+msg.Message),
			tea.Quit,
		)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m spinnerModel) View() string {
	switch m.phase {
	case phaseStarting:
		line := m.spinner.View() + " Starting pod " + m.podName
		if m.currentEvent != "" {
			line += "\n  → " + m.currentEvent
		}
		return line
	case phaseAttaching:
		return m.spinner.View() + " Attaching to " + m.containerName + "..."
	default:
		return ""
	}
}

// RunStartupSpinner runs the spinner UI and blocks until done.
// setup is called with the tea.Program so the caller can start
// goroutines that send messages via p.Send().
func RunStartupSpinner(podName, containerName string, isNewPod bool, setup func(p *tea.Program)) StartupResult {
	model := newSpinnerModel(podName, containerName, isNewPod)
	p := tea.NewProgram(model, tea.WithoutSignals())

	setup(p)

	result, err := p.Run()
	if err != nil {
		return StartupResult{Error: err}
	}

	final := result.(spinnerModel)
	return StartupResult{
		Pod:       final.readyPod,
		Cancelled: final.cancelled,
		Error:     final.err,
	}
}
