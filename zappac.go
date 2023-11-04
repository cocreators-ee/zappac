package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jeandeaual/go-locale"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cocreators-ee/zappaclang"
	"github.com/muesli/termenv"
)

const maxHistory = 10

var (
	color       = termenv.EnvColorProfile().Color
	bad         = termenv.Style{}.Foreground(color("198")).Styled
	variable    = termenv.Style{}.Foreground(color("39")).Styled
	op          = termenv.Style{}.Foreground(color("243")).Styled
	value       = termenv.Style{}.Foreground(color("87")).Styled
	placeholder = termenv.Style{}.Foreground(color("75")).Styled
	function    = termenv.Style{}.Foreground(color("156")).Styled
	unparsed    = termenv.Style{}.Foreground(color("247")).Styled
	current     = termenv.Style{}.Underline().Styled
)

var printer *message.Printer = nil

type historyItem struct {
	input  string
	result string
}

type model struct {
	zs                *zappaclang.ZappacState
	input             string
	result            string
	err               error
	history           []historyItem
	cursor            int
	placeholderResult string
	parsedNodes       []zappaclang.Node
}

func main() {
	userLanguage, err := locale.GetLanguage()
	if err != nil {
		userLanguage = "en"
	}

	printer = message.NewPrinter(language.Make(userLanguage))

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func initialModel() model {
	return model{
		zs:          zappaclang.NewZappacState(""),
		input:       "",
		result:      "",
		cursor:      -1,
		history:     []historyItem{},
		parsedNodes: []zappaclang.Node{},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) appendHistory(input string, result string) {
	m.history = append([]historyItem{{m.input, result}}, m.history...)
	if len(m.history) > maxHistory {
		m.history = m.history[:maxHistory]
	}
}

func (m *model) exec(preview bool) {
	nodes, err := zappaclang.Parse(m.input)
	m.parsedNodes = nodes
	if err != nil {
		m.err = err
	} else {
		result, err := m.zs.Exec(nodes, !preview)
		if err != nil {
			m.err = err
		} else {
			if preview {
				m.placeholderResult = result
				m.err = nil
			} else {
				m.result = result
				m.err = nil
				m.appendHistory(m.input, result)
				m.input = ""
				m.cursor = -1
				m.placeholderResult = ""
				m.parsedNodes = []zappaclang.Node{}

				if nodes[0].Type() == zappaclang.NodeClear {
					m.history = []historyItem{}
				}
			}
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
			m.exec(false)
		case "up":
			if m.cursor < len(m.history)-1 {
				m.cursor++
			}
			if len(m.history) > 0 {
				m.input = m.history[m.cursor].input
			}
		case "down":
			if m.cursor > -1 {
				m.cursor--
			}

			if m.cursor >= 0 {
				m.input = m.history[m.cursor].input
			} else {
				m.input = ""
			}
		case "backspace":
			if len(m.input) > 0 {
				// Remove last (potentially) multi-byte character
				r, size := utf8.DecodeLastRuneInString(m.input)
				if r == utf8.RuneError && (size == 0 || size == 1) {
					size = 0
				}
				m.input = m.input[:len(m.input)-size]
				m.exec(true)
			}
		default:
			for _, r := range msg.Runes {
				m.input += string(r)
			}
			m.exec(true)
		}
	}

	return m, nil
}

func formatInput(input string, parsedNodes []zappaclang.Node) string {
	result := ""
	lastPos := zappaclang.Pos(0)

	for idx, node := range parsedNodes {
		typ := node.Type()
		if typ == zappaclang.NodeEOF {
			break
		}

		start := node.Position()
		end := start

		if len(parsedNodes) > idx+1 {
			end = parsedNodes[idx+1].Position()
			maxEnd := zappaclang.Pos(len(input))
			if end > maxEnd {
				end = maxEnd
			}
			if end < 0 {
				end = 0
			}
		}

		val := input[start:end]
		if typ == zappaclang.NodeNumber {
			val = numberFormat(val)
		}
		if zappaclang.IsNodeType(node, zappaclang.ValueNodes) {
			val = value(val)
		}
		if zappaclang.IsNodeType(node, zappaclang.OperatorNodes) {
			val = op(val)
		}
		if zappaclang.IsNodeType(node, zappaclang.FunctionNodes) {
			val = function(val)
		}

		result += val
		lastPos = end
	}

	if lastPos <= zappaclang.Pos(len(input)-1) {
		result += unparsed(input[lastPos:])
	}

	return result
}

func numberFormat(value string) string {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}

	// TODO: This seems to round off decimals rather heavily
	return printer.Sprintf("%v", number.Decimal(val))
}

func (m model) View() string {
	v := "> " + formatInput(m.input, m.parsedNodes) + "\n"

	if m.err != nil {
		v += "\n"
		v += bad(m.err.Error()) + "\n"
	}

	if m.placeholderResult != "" {
		v += placeholder(numberFormat(m.placeholderResult)) + "\n"
	}

	if m.result != "" {
		v += value(numberFormat(m.result)) + "\n"
	}

	if len(m.zs.Variables) > 0 {
		v += "\n"

		// Sort keys
		keys := []string{}
		for key := range m.zs.Variables {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			num := m.zs.Variables[key].String()
			v += fmt.Sprintf("%s %s %s\n", variable(key), op("="), value(numberFormat(num)))
		}
	}

	if len(m.history) > 0 {
		v += "\n"

		for i := len(m.history) - 1; i >= 0; i-- {
			item := m.history[i]
			line := ""
			if strings.Contains(item.input, "=") {
				line = placeholder(item.input)
			} else {
				line = fmt.Sprintf("%s %s %s", placeholder(item.input), op("="), value(numberFormat(item.result)))
			}

			if i == m.cursor {
				line = current(line)
			}

			v += line + "\n"
		}
	}

	return v
}
