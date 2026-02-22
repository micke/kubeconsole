package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Option represents a selectable item with a display label and underlying value.
type Option[T any] struct {
	Label string
	Value T
}

// Select displays an interactive select prompt with inline type-to-filter.
// Returns the selected option's value, or an error if the user cancels.
func Select[T any](title string, options []Option[T]) (T, error) {
	m := selectModel[T]{
		title:   title,
		options: options,
		height:  10,
	}
	m.applyFilter()

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		var zero T
		return zero, err
	}

	final := result.(selectModel[T])
	if !final.chosen {
		var zero T
		return zero, fmt.Errorf("cancelled")
	}

	idx := final.filtered[final.cursor]
	return final.options[idx].Value, nil
}

type selectModel[T any] struct {
	title       string
	options     []Option[T]
	filtered    []int
	cursor      int
	filter      string
	height      int
	offset      int
	chosen      bool
	chosenLabel string
}

func (m selectModel[T]) Init() tea.Cmd {
	return nil
}

func (m selectModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				m.chosen = true
				m.chosenLabel = m.options[idx].Label
				return m, tea.Sequence(
					tea.Println(GreenCheck+" "+m.title+" "+selectedStyle.Render(m.chosenLabel)),
					tea.Quit,
				)
			}
			return m, tea.Quit

		case tea.KeyUp, tea.KeyCtrlP, tea.KeyCtrlK:
			if len(m.filtered) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.filtered) - 1
				}
				m.adjustOffset()
			}
			return m, nil

		case tea.KeyDown, tea.KeyCtrlN, tea.KeyCtrlJ:
			if len(m.filtered) > 0 {
				m.cursor++
				if m.cursor >= len(m.filtered) {
					m.cursor = 0
				}
				m.adjustOffset()
			}
			return m, nil

		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
			return m, nil

		case tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.applyFilter()
			return m, nil
		}
	}

	return m, nil
}

func (m *selectModel[T]) applyFilter() {
	m.filtered = m.filtered[:0]
	query := strings.ToLower(m.filter)
	for i, opt := range m.options {
		if query == "" || strings.Contains(strings.ToLower(opt.Label), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
	m.offset = 0
}

func (m *selectModel[T]) adjustOffset() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	filterStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	normalStyle   = lipgloss.NewStyle()
)

func (m selectModel[T]) View() string {
	if m.chosen {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render(m.title))
	b.WriteByte('\n')

	if m.filter != "" {
		b.WriteString(filterStyle.Render("> " + m.filter))
		b.WriteByte('\n')
	}

	if len(m.filtered) == 0 {
		b.WriteString(filterStyle.Render("  No matching options"))
		b.WriteByte('\n')
		return b.String()
	}

	end := m.offset + m.height
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.offset; i < end; i++ {
		idx := m.filtered[i]
		label := m.options[idx].Label

		if i == m.cursor {
			b.WriteString(selectedStyle.Render("  ▸ " + label))
		} else {
			b.WriteString(normalStyle.Render("    " + label))
		}
		b.WriteByte('\n')
	}

	return b.String()
}
