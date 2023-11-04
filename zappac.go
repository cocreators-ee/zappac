package main

import (
	"fmt"
	"os"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cocreators-ee/zappaclang"
	"github.com/muesli/termenv"
)

var (
	color    = termenv.EnvColorProfile().Color
	bad      = termenv.Style{}.Foreground(color("198")).Styled
	variable = termenv.Style{}.Foreground(color("39")).Styled
	op       = termenv.Style{}.Foreground(color("243")).Styled
)

type model struct {
	zs      *zappaclang.ZappacState
	input   string
	result  string
	err     error
	history []string
	cursor  int
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func initialModel() model {
	return model{
		zs:      zappaclang.NewZappacState(""),
		input:   "",
		result:  "",
		cursor:  -1,
		history: []string{},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) exec() {
	nodes, err := zappaclang.Parse(m.input)
	if err != nil {
		m.err = err
	} else {
		result, err := m.zs.Exec(nodes, true)
		if err != nil {
			m.err = err
		} else {
			m.result = result
			m.err = nil
			m.history = append([]string{m.input}, m.history...)
			m.input = ""
			m.cursor = -1
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m.exec()
		case "up":
			if m.cursor < len(m.history)-1 {
				m.cursor++
			}
			m.input = m.history[m.cursor]
		case "down":
			if m.cursor > -1 {
				m.cursor--
			}

			if m.cursor >= 0 {
				m.input = m.history[m.cursor]
			} else {
				m.input = ""
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			// TODO: Filter
			m.input += msg.String()
		}
	}

	return m, nil
}

func (m model) View() string {
	v := "> " + m.input

	if m.result != "" {
		v += "\n"
		v += m.result
	}

	if m.err != nil {
		v += "\n"
		v += bad(m.err.Error())
	}

	if len(m.zs.Variables) > 0 {
		v += "\n"
		v += "\n"

		// Sort keys
		keys := []string{}
		for key := range m.zs.Variables {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			num := m.zs.Variables[key]
			v += fmt.Sprintf("%s %s %s\n", variable(key), op("="), num)
		}
	}

	return v
}
