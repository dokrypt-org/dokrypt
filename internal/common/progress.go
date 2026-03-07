package common

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"
)

type Output interface {
	Info(msg string, args ...any)
	Success(msg string, args ...any)
	Warning(msg string, args ...any)
	Error(msg string, args ...any)
	Table(headers []string, rows [][]string)
	Progress(total int) ProgressBar
	Spinner(msg string) Spinner
	JSON(v any)
	Step(current, total int, msg string)
}

type ProgressBar interface {
	Increment()
	SetCurrent(n int)
	Done()
}

type Spinner interface {
	Update(msg string)
	Stop()
	StopWithSuccess(msg string)
	StopWithError(msg string)
}

type cliStyles struct {
	infoBadge    lipgloss.Style
	successBadge lipgloss.Style
	warningBadge lipgloss.Style
	errorBadge   lipgloss.Style
	stepBadge    lipgloss.Style
	tableHeader  lipgloss.Style
	tableCell    lipgloss.Style
	tableBorder  lipgloss.Style
	spinner      lipgloss.Style
}

func newStyles(r *lipgloss.Renderer) cliStyles {
	return cliStyles{
		infoBadge:    r.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4")).Padding(0, 1),
		successBadge: r.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("2")).Padding(0, 1),
		warningBadge: r.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3")).Padding(0, 1),
		errorBadge:   r.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("1")).Padding(0, 1),
		stepBadge:    r.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("6")).Padding(0, 1),
		tableHeader:  r.NewStyle().Bold(true).Foreground(lipgloss.Color("4")).Padding(0, 1),
		tableCell:    r.NewStyle().Padding(0, 1),
		tableBorder:  r.NewStyle().Foreground(lipgloss.Color("8")),
		spinner:      r.NewStyle().Foreground(lipgloss.Color("6")),
	}
}

type ConsoleOutput struct {
	writer   io.Writer
	noColor  bool
	quiet    bool
	renderer *lipgloss.Renderer
	styles   cliStyles
}

func NewConsoleOutput(writer io.Writer, noColor, quiet bool) *ConsoleOutput {
	if writer == nil {
		writer = os.Stdout
	}
	nc := noColor || NoColor()

	r := lipgloss.NewRenderer(writer)
	if nc {
		r.SetColorProfile(termenv.Ascii)
	} else {
		r.SetColorProfile(termenv.ANSI)
	}

	return &ConsoleOutput{
		writer:   writer,
		noColor:  nc,
		quiet:    quiet,
		renderer: r,
		styles:   newStyles(r),
	}
}

func (o *ConsoleOutput) Info(msg string, args ...any) {
	if o.quiet {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintln(o.writer, o.styles.infoBadge.Render("i")+" "+formatted)
}

func (o *ConsoleOutput) Success(msg string, args ...any) {
	if o.quiet {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintln(o.writer, o.styles.successBadge.Render("ok")+" "+formatted)
}

func (o *ConsoleOutput) Warning(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintln(o.writer, o.styles.warningBadge.Render("!")+" "+formatted)
}

func (o *ConsoleOutput) Error(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintln(o.writer, o.styles.errorBadge.Render("x")+" "+formatted)
}

func (o *ConsoleOutput) Step(current, total int, msg string) {
	if o.quiet {
		return
	}
	prefix := fmt.Sprintf("%d/%d", current, total)
	fmt.Fprintln(o.writer, o.styles.stepBadge.Render(prefix)+" "+msg)
}

func (o *ConsoleOutput) Table(headers []string, rows [][]string) {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(o.styles.tableBorder).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		BorderHeader(true).
		BorderColumn(true).
		BorderRow(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return o.styles.tableHeader
			}
			return o.styles.tableCell
		})

	fmt.Fprintln(o.writer, t.Render())
}

func (o *ConsoleOutput) Progress(total int) ProgressBar {
	return &consoleProgressBar{
		output: o,
		total:  total,
	}
}

func (o *ConsoleOutput) Spinner(msg string) Spinner {
	s := &consoleSpinner{
		output:  o,
		msg:     msg,
		done:    make(chan struct{}),
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
	if !o.quiet {
		go s.run()
	}
	return s
}

func (o *ConsoleOutput) JSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		o.Error("Failed to marshal JSON: %v", err)
		return
	}
	fmt.Fprintln(o.writer, string(data))
}

type consoleProgressBar struct {
	output  *ConsoleOutput
	total   int
	current int
}

func (p *consoleProgressBar) Increment() {
	p.current++
	p.render()
}

func (p *consoleProgressBar) SetCurrent(n int) {
	p.current = n
	p.render()
}

func (p *consoleProgressBar) Done() {
	p.current = p.total
	p.render()
	fmt.Fprintln(p.output.writer)
}

func (p *consoleProgressBar) render() {
	if p.output.quiet {
		return
	}
	width := 30
	filled := 0
	if p.total > 0 {
		filled = (p.current * width) / p.total
	}
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := 0
	if p.total > 0 {
		pct = (p.current * 100) / p.total
	}
	fmt.Fprintf(p.output.writer, "\r  %s %d%% (%d/%d)", bar, pct, p.current, p.total)
}

type consoleSpinner struct {
	output *ConsoleOutput
	msg    string
	mu     sync.Mutex
	done   chan struct{}
	once   sync.Once
	frames []string
}

func (s *consoleSpinner) run() {
	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.msg
			s.mu.Unlock()
			frame := s.output.styles.spinner.Render(s.frames[i%len(s.frames)])
			fmt.Fprintf(s.output.writer, "\r  %s %s", frame, msg)
			i++
		}
	}
}

func (s *consoleSpinner) Update(msg string) {
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

func (s *consoleSpinner) Stop() {
	s.once.Do(func() {
		close(s.done)
		fmt.Fprint(s.output.writer, "\r\033[K") // Clear line
	})
}

func (s *consoleSpinner) StopWithSuccess(msg string) {
	s.once.Do(func() {
		close(s.done)
		fmt.Fprint(s.output.writer, "\r\033[K")
		s.output.Success("%s", msg)
	})
}

func (s *consoleSpinner) StopWithError(msg string) {
	s.once.Do(func() {
		close(s.done)
		fmt.Fprint(s.output.writer, "\r\033[K")
		s.output.Error("%s", msg)
	})
}

type JSONOutput struct {
	writer io.Writer
}

func NewJSONOutput(writer io.Writer) *JSONOutput {
	if writer == nil {
		writer = os.Stdout
	}
	return &JSONOutput{writer: writer}
}

type jsonMessage struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func (o *JSONOutput) emit(level, msg string, args ...any) {
	data, err := json.Marshal(jsonMessage{
		Level:   level,
		Message: fmt.Sprintf(msg, args...),
	})
	if err != nil {
		fmt.Fprintf(o.writer, `{"level":%q,"message":%q}`+"\n", level, fmt.Sprintf(msg, args...))
		return
	}
	fmt.Fprintln(o.writer, string(data))
}

func (o *JSONOutput) Info(msg string, args ...any)    { o.emit("info", msg, args...) }

func (o *JSONOutput) Success(msg string, args ...any) { o.emit("success", msg, args...) }

func (o *JSONOutput) Warning(msg string, args ...any) { o.emit("warning", msg, args...) }

func (o *JSONOutput) Error(msg string, args ...any)   { o.emit("error", msg, args...) }

func (o *JSONOutput) Step(current, total int, msg string) {
	data, err := json.Marshal(map[string]any{
		"level":   "step",
		"current": current,
		"total":   total,
		"message": msg,
	})
	if err != nil {
		fmt.Fprintf(o.writer, `{"level":"step","current":%d,"total":%d,"message":%q}`+"\n", current, total, msg)
		return
	}
	fmt.Fprintln(o.writer, string(data))
}

func (o *JSONOutput) Table(headers []string, rows [][]string) {
	result := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		m := make(map[string]string)
		for i, h := range headers {
			if i < len(row) {
				m[h] = row[i]
			}
		}
		result = append(result, m)
	}
	o.JSON(result)
}

func (o *JSONOutput) Progress(_ int) ProgressBar { return &noopProgressBar{} }

func (o *JSONOutput) Spinner(_ string) Spinner { return &noopSpinner{} }

func (o *JSONOutput) JSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(o.writer, `{"error":"failed to marshal JSON: %s"}`+"\n", err)
		return
	}
	fmt.Fprintln(o.writer, string(data))
}

type noopProgressBar struct{}

func (p *noopProgressBar) Increment()      {}
func (p *noopProgressBar) SetCurrent(_ int) {}
func (p *noopProgressBar) Done()           {}

type noopSpinner struct{}

func (s *noopSpinner) Update(_ string)          {}
func (s *noopSpinner) Stop()                    {}
func (s *noopSpinner) StopWithSuccess(_ string) {}
func (s *noopSpinner) StopWithError(_ string)   {}
