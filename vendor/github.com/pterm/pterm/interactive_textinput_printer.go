package pterm

import (
	"strings"

	"atomicgo.dev/cursor"
	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/mattn/go-runewidth"

	"github.com/pterm/pterm/internal"
)

// DefaultInteractiveTextInput is the default InteractiveTextInput printer.
var DefaultInteractiveTextInput = InteractiveTextInputPrinter{
	DefaultText: "Input text",
	Delimiter:   ": ",
	TextStyle:   &ThemeDefault.PrimaryStyle,
	Mask:        "",
}

// InteractiveTextInputPrinter is a printer for interactive select menus.
type InteractiveTextInputPrinter struct {
	TextStyle       *Style
	DefaultText     string
	DefaultValue    string
	Delimiter       string
	MultiLine       bool
	Mask            string
	OnInterruptFunc func()

	input         []string
	cursorXPos    int
	cursorYPos    int
	text          string
	startedTyping bool
	valueStyle    *Style
}

// WithDefaultText sets the default text.
func (p InteractiveTextInputPrinter) WithDefaultText(text string) *InteractiveTextInputPrinter {
	p.DefaultText = text
	return &p
}

// WithDefaultValue sets the default value.
func (p InteractiveTextInputPrinter) WithDefaultValue(value string) *InteractiveTextInputPrinter {
	p.DefaultValue = value
	return &p
}

// WithTextStyle sets the text style.
func (p InteractiveTextInputPrinter) WithTextStyle(style *Style) *InteractiveTextInputPrinter {
	p.TextStyle = style
	return &p
}

// WithMultiLine sets the multi line flag.
func (p InteractiveTextInputPrinter) WithMultiLine(multiLine ...bool) *InteractiveTextInputPrinter {
	p.MultiLine = internal.WithBoolean(multiLine)
	return &p
}

// WithMask sets the mask.
func (p InteractiveTextInputPrinter) WithMask(mask string) *InteractiveTextInputPrinter {
	p.Mask = mask
	return &p
}

// WithOnInterruptFunc sets the function to execute on exit of the input reader
func (p InteractiveTextInputPrinter) WithOnInterruptFunc(exitFunc func()) *InteractiveTextInputPrinter {
	p.OnInterruptFunc = exitFunc
	return &p
}

// WithDelimiter sets the delimiter between the message and the input.
func (p InteractiveTextInputPrinter) WithDelimiter(delimiter string) *InteractiveTextInputPrinter {
	p.Delimiter = delimiter
	return &p
}

// Show shows the interactive select menu and returns the selected entry.
func (p InteractiveTextInputPrinter) Show(text ...string) (string, error) {
	// should be the first defer statement to make sure it is executed last
	// and all the needed cleanup can be done before
	cancel, exit := internal.NewCancelationSignal(p.OnInterruptFunc)
	defer exit()

	var areaText string

	if len(text) == 0 || text[0] == "" {
		text = []string{p.DefaultText}
	}

	if p.MultiLine {
		areaText = p.TextStyle.Sprintfln("%s %s %s", text[0], ThemeDefault.SecondaryStyle.Sprint("[Press tab to submit]"), p.Delimiter)
	} else {
		areaText = p.TextStyle.Sprintf("%s%s", text[0], p.Delimiter)
	}

	p.text = areaText
	area := cursor.NewArea()
	area.Update(areaText)
	area.StartOfLine()

	if !p.MultiLine {
		cursor.Right(runewidth.StringWidth(RemoveColorFromString(areaText)))
	}

	if p.DefaultValue != "" {
		p.input = append(p.input, p.DefaultValue)
		p.updateArea(&area)
	}

	err := keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		if !p.MultiLine {
			p.cursorYPos = 0
		}
		if len(p.input) == 0 {
			p.input = append(p.input, "")
		}

		switch key.Code {
		case keys.Tab:
			if p.MultiLine {
				area.Bottom()
				return true, nil
			}
		case keys.Enter:
			if p.DefaultValue != "" && !p.startedTyping {
				if p.MultiLine {
					area.Bottom()
				}
				return true, nil
			}

			if p.MultiLine {
				if key.AltPressed {
					p.cursorXPos = 0
				}
				line := p.input[p.cursorYPos]
				width := internal.GetStringMaxWidth(line)
				pos := width + p.cursorXPos
				idx := runeIndexAtColumn(line, pos)
				runes := []rune(line)
				appendAfterY := append([]string{}, p.input[p.cursorYPos+1:]...)
				p.input[p.cursorYPos] = string(runes[:idx])
				p.input = append(p.input[:p.cursorYPos+1], string(runes[idx:]))
				p.input = append(p.input, appendAfterY...)
				p.cursorYPos++
				p.cursorXPos = -internal.GetStringMaxWidth(p.input[p.cursorYPos])
				cursor.StartOfLine()
			} else {
				return true, nil
			}
		case keys.RuneKey:
			if !p.startedTyping {
				p.startedTyping = true
			}
			line := p.input[p.cursorYPos]
			width := internal.GetStringMaxWidth(line)
			pos := width + p.cursorXPos
			idx := runeIndexAtColumn(line, pos)
			runes := append([]rune{}, []rune(line)...)
			newRunes := append(append(runes[:idx], []rune(key.String())...), runes[idx:]...)
			p.input[p.cursorYPos] = string(newRunes)
		case keys.Space:
			if !p.startedTyping {
				p.startedTyping = true
			}
			line := p.input[p.cursorYPos]
			width := internal.GetStringMaxWidth(line)
			pos := width + p.cursorXPos
			idx := runeIndexAtColumn(line, pos)
			runes := append([]rune{}, []rune(line)...)
			newRunes := append(append(runes[:idx], ' '), runes[idx:]...)
			p.input[p.cursorYPos] = string(newRunes)
		case keys.Backspace:
			if !p.startedTyping {
				p.startedTyping = true
			}
			line := p.input[p.cursorYPos]
			width := internal.GetStringMaxWidth(line)
			pos := width + p.cursorXPos
			idx := runeIndexAtColumn(line, pos)
			if idx == 0 {
				if p.cursorYPos > 0 {
					p.input[p.cursorYPos-1] += p.input[p.cursorYPos]
					p.input = append(p.input[:p.cursorYPos], p.input[p.cursorYPos+1:]...)
					p.cursorXPos = 0
					p.cursorYPos--
				}
			} else {
				newLine, _ := deleteRuneAtIndex(line, idx-1)
				p.input[p.cursorYPos] = newLine
			}
		case keys.Delete:
			if !p.startedTyping {
				p.input = []string{""}
				p.startedTyping = true
				return false, nil
			}
			line := p.input[p.cursorYPos]
			width := internal.GetStringMaxWidth(line)
			pos := width + p.cursorXPos
			idx := runeIndexAtColumn(line, pos)
			runes := []rune(line)
			if idx >= len(runes) {
				if p.cursorYPos < len(p.input)-1 {
					p.input[p.cursorYPos] += p.input[p.cursorYPos+1]
					p.input = append(p.input[:p.cursorYPos+1], p.input[p.cursorYPos+2:]...)
					p.cursorXPos = 0
				}
			} else {
				newLine, _ := deleteRuneAtIndex(line, idx)
				p.input[p.cursorYPos] = newLine
			}
		case keys.CtrlC:
			cancel()
			return true, nil
		case keys.Down:
			if !p.MultiLine {
				return false, nil
			}
			if !p.startedTyping {
				p.input = []string{""}
				p.startedTyping = true
			}
			if p.cursorYPos+1 < len(p.input) {
				p.cursorXPos = (internal.GetStringMaxWidth(p.input[p.cursorYPos]) + p.cursorXPos) - internal.GetStringMaxWidth(p.input[p.cursorYPos+1])
				if p.cursorXPos > 0 {
					p.cursorXPos = 0
				}
				p.cursorYPos++
			}
		case keys.Up:
			if !p.MultiLine {
				return false, nil
			}
			if !p.startedTyping {
				p.input = []string{""}
				p.startedTyping = true
			}
			if p.cursorYPos > 0 {
				p.cursorXPos = (internal.GetStringMaxWidth(p.input[p.cursorYPos]) + p.cursorXPos) - internal.GetStringMaxWidth(p.input[p.cursorYPos-1])
				if p.cursorXPos > 0 {
					p.cursorXPos = 0
				}
				p.cursorYPos--
			}
		}

		if internal.GetStringMaxWidth(p.input[p.cursorYPos]) > 0 {
			line := p.input[p.cursorYPos]
			width := internal.GetStringMaxWidth(line)
			pos := width + p.cursorXPos
			idx := runeIndexAtColumn(line, pos)
			runes := []rune(line)
			switch key.Code {
			case keys.Right:
				if idx < len(runes) {
					endCol := runeStartColumnForIndex(line, idx) + runewidth.RuneWidth(runes[idx])
					p.cursorXPos = endCol - width
				} else if p.cursorYPos < len(p.input)-1 {
					p.cursorYPos++
					p.cursorXPos = -internal.GetStringMaxWidth(p.input[p.cursorYPos])
				}
			case keys.Left:
				if idx > 0 {
					newPos := runeStartColumnForIndex(line, idx-1)
					p.cursorXPos = newPos - width
				} else if p.cursorYPos > 0 {
					p.cursorYPos--
					p.cursorXPos = 0
				}
			}
		}

		p.updateArea(&area)

		return false, nil
	})
	if err != nil {
		return "", err
	}

	// Add new line
	Println()

	if !p.startedTyping {
		return p.DefaultValue, nil
	}

	return strings.Join(p.input, "\n"), nil
}

func (p InteractiveTextInputPrinter) updateArea(area *cursor.Area) string {
	if !p.MultiLine {
		p.cursorYPos = 0
	}
	areaText := p.text

	for i, s := range p.input {
		displayS := s
		if !p.MultiLine && !p.startedTyping && p.DefaultValue != "" && i == 0 && s == p.DefaultValue {
			displayS = Gray(s)
		}
		if i < len(p.input)-1 {
			areaText += displayS + "\n"
		} else {
			areaText += displayS
		}
	}

	if p.Mask != "" {
		areaText = p.text + strings.Repeat(p.Mask, internal.GetStringMaxWidth(areaText)-internal.GetStringMaxWidth(p.text))
	}

	if p.cursorXPos+internal.GetStringMaxWidth(p.input[p.cursorYPos]) < 1 {
		p.cursorXPos = -internal.GetStringMaxWidth(p.input[p.cursorYPos])
	}

	area.Update(areaText)
	area.Top()
	area.Down(p.cursorYPos + 1)
	area.StartOfLine()
	if p.MultiLine {
		cursor.Right(internal.GetStringMaxWidth(p.input[p.cursorYPos]) + p.cursorXPos)
	} else {
		cursor.Right(internal.GetStringMaxWidth(areaText) + p.cursorXPos)
	}
	return areaText
}

func runeIndexAtColumn(s string, col int) int {
	runes := []rune(s)
	var colIdx int
	for i, r := range runes {
		w := runewidth.RuneWidth(r)
		if colIdx+w > col {
			return i
		}
		colIdx += w
	}
	return len(runes)
}

func runeStartColumnForIndex(s string, idx int) int {
	runes := []rune(s)
	if idx <= 0 {
		return 0
	}
	var colIdx int
	for i := 0; i < idx && i < len(runes); i++ {
		colIdx += runewidth.RuneWidth(runes[i])
	}
	return colIdx
}

func deleteRuneAtIndex(s string, idx int) (newS string, removedWidth int) {
	runes := []rune(s)
	if idx < 0 || idx >= len(runes) {
		return s, 0
	}
	w := runewidth.RuneWidth(runes[idx])
	newRunes := append(runes[:idx], runes[idx+1:]...)
	return string(newRunes), w
}
