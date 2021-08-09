package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/c-bata/go-prompt"
	goterm "golang.org/x/term"
)

func rootdir() string { return filepath.Join(homedir(), ".gopherd") }
func history() string { return filepath.Join(rootdir(), "history") }
func profile() string { return filepath.Join(rootdir(), "profile") }
func logfile() string { return filepath.Join(rootdir(), "log.txt") }

func homedir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	return "/"
}

func loadHistory() []string {
	f, err := os.Open(history())
	if err != nil {
		return nil
	}
	defer f.Close()
	var (
		s = bufio.NewScanner(f)
		h []string
	)
	for s.Scan() {
		h = append(h, s.Text())
	}
	return h
}

func saveHistory(h []string) {
	if os.MkdirAll(rootdir(), 0755) != nil {
		return
	}
	f, err := os.Create(history())
	if err != nil {
		return
	}
	defer f.Close()
	for i := range h {
		fmt.Fprintln(f, h[i])
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error)       { return len(p), nil }
func (discard) WriteString(s string) (int, error) { return len(s), nil }
func (discard) Close() error                      { return nil }

// stRingBuffer is a ring buffer of strings.
type stRingBuffer struct {
	// entries contains max elements.
	entries []string
	max     int
	// head contains the index of the element most recently added to the ring.
	head int
	// size contains the number of elements in the ring.
	size int
}

func (s *stRingBuffer) init() {
	const defaultNumEntries = 1000
	s.entries = make([]string, defaultNumEntries)
	s.max = defaultNumEntries
}

func (s *stRingBuffer) add(a string) {
	s.head = (s.head + 1) % s.max
	s.entries[s.head] = a
	if s.size < s.max {
		s.size++
	}
}

func (s *stRingBuffer) all() []string {
	var h = make([]string, s.size)
	for i := 0; i < s.size; i++ {
		index := s.head - i
		if index < 0 {
			index += s.max
		}
		h[s.size-1-i] = s.entries[index]
	}
	return h
}

type terminal struct {
	prompt *prompt.Prompt
	env    *enviroment
	over   bool
}

func newTerminal() *terminal {
	term := &terminal{
		env: newEnviroment(),
	}
	h := loadHistory()
	if f, err := os.Create(logfile()); err != nil {
		term.env.logfile = discard{}
	} else {
		term.env.logfile = f
	}
	for i := range h {
		term.env.history.add(h[i])
	}
	term.prompt = prompt.New(term.exec, term.complete,
		prompt.OptionInputTextColor(prompt.DefaultColor),
		prompt.OptionLivePrefix(term.ps1),
		prompt.OptionSetExitCheckerOnInput(term.checkExit),
		prompt.OptionPrefixTextColor(prompt.DefaultColor),
		prompt.OptionHistory(h),
	)
	return term
}

func (term *terminal) run() error {
	if err := term.env.init(); err != nil {
		return err
	}
	// load profile
	stdin, stdout := term.env.stdin, term.env.stdout
	term.env.nofork = true
	if filename := profile(); filename != "" {
		if _, err := os.Stat(filename); err == nil {
			term.env.run(context.Background(), fmt.Sprintf("run %q", filename), nil, io.Discard)
		}
	}
	term.env.nofork = false
	term.env.stdin, term.env.stdout = stdin, stdout
	term.env.println("Welcome to gopherd-cli, run help to show help information, run exit/quit to exit.")

	stdinFD := int(os.Stdin.Fd())
	stdinState, err := goterm.GetState(stdinFD)
	if err != nil {
		return err
	}
	defer func() {
		if x := recover(); x != nil {
			if e, ok := x.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", x)
			}
		}
		goterm.Restore(stdinFD, stdinState)
		term.env.shutdown()
	}()

	term.prompt.Run()
	saveHistory(term.env.history.all())
	return nil
}

func (term *terminal) exec(line string) {
	if term.checkExit(line, true) {
		term.env.history.add(line)
		return
	}
	line = strings.TrimSpace(line)
	term.env.stdout = os.Stdout
	if term.env.more.size > 0 {
		line = strings.ToLower(line)
		if line == "y" || line == "yes" {
			term.env.stdout.Write(term.env.more.content.Bytes())
		}
		term.env.more.content.Reset()
		term.env.more.size = 0
		return
	}
	if line == "" {
		return
	}
	term.env.history.add(line)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	term.env.run(ctx, line, strings.NewReader(""), os.Stdout)
}

func (term *terminal) ps1() (string, bool) {
	if term.over {
		return "", true
	}
	return term.env.ps1(), true
}

func (term *terminal) complete(d prompt.Document) []prompt.Suggest {
	var (
		runes []rune
		end   = -1
	)
	for _, r := range d.Text {
		isSpace := unicode.IsSpace(r)
		if len(runes) > 0 || !isSpace {
			if isSpace && end < 0 {
				end = len(runes)
			}
			runes = append(runes, r)
		}
	}
	if len(runes) == 0 {
		return nil
	}
	var (
		text     = string(runes)
		suggests []prompt.Suggest
	)
	if end > 0 {
		// (TODO)
		//cmd, ok := commands[string(runes[:end])]
		//if !ok || cmd.complete == nil {
		//	return nil
		//}
		//var pos int
		//args, err := shell.Split(text)
		//if err != nil {
		//	ie, ok := err.(*shell.IncompleteError)
		//	if !ok {
		//		return nil
		//	}
		//	pos = ie.Begin
		//} else {
		//	pos = len(text)
		//}
		//appended := cmd.complete(text, pos, args)
		//for i := range appended {
		//	suggests = append(suggests, prompt.Suggest{
		//		Text: text + appended[i],
		//	})
		//}
	}

	for _, name := range commandTrie.Search(text, -1) {
		cmd, ok := commands[name]
		if !ok {
			if term.env.redis != nil && term.env.redis.commands != nil {
				if _, ok := term.env.redis.commands[strings.ToLower(name)]; !ok {
					continue
				}
				if tip, ok := redisTips[name]; ok {
					desc := tip.syntax
					if tip.usage != "" {
						desc += " (" + tip.usage + ")"
					}
					suggests = append(suggests, prompt.Suggest{
						Text:        name,
						Description: desc,
					})
				} else {
					suggests = append(suggests, prompt.Suggest{
						Text:        name,
						Description: "redis command",
					})
				}
			}
			continue
		}
		suggests = append(suggests, prompt.Suggest{
			Text:        name,
			Description: cmd.usage,
		})
	}
	return suggests
}

func (term *terminal) checkExit(in string, breakline bool) bool {
	if !breakline {
		return false
	}
	if !term.over {
		in = strings.ToLower(strings.TrimSpace(in))
		term.over = in == "quit" || in == "exit"
	}
	return term.over
}

type exitError struct {
	reason string
	code   int
}

func (e *exitError) Error() string { return e.reason }
func (e *exitError) ExitCode() int { return e.code }
