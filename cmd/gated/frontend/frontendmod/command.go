package frontendmod

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/gopherd/doge/proto"
)

type command struct {
	name   string
	format string
	usage  string
	run    func(*frontendModule, *session, []string) error
}

var (
	commands = make(map[string]*command)
)

func register(cmd *command) {
	if cmd.name == "" {
		panic("register a empty command")
	}
	if _, dup := commands[cmd.name]; dup {
		panic("register called twice for " + cmd.name)
	}
	commands[cmd.name] = cmd
}

func init() {
	// .help [command]
	register(&command{
		name:   "help",
		format: "[command]",
		usage:  "show help information",
		run: func(f *frontendModule, sess *session, args []string) error {
			var (
				cmds []*command
			)
			if len(args) > 0 {
				for i := range args {
					cmd := commands[strings.ToLower(args[i])]
					if cmd == nil {
						return errorln(sess, "command ", args[i], " not found")
					}
					cmds = append(cmds, cmd)
				}
			} else {
				for _, cmd := range commands {
					cmds = append(cmds, cmd)
				}
				sort.Slice(cmds, func(i, j int) bool {
					return cmds[i].name < cmds[j].name
				})
			}
			p := getPrinter()
			for _, cmd := range cmds {
				if cmd.format != "" {
					p.println("."+cmd.name+" ", cmd.format)
				} else {
					p.println("." + cmd.name)
				}
				p.println("\t" + cmd.usage)
			}
			return p.flush(sess)
		},
	})

	// .ping [content]
	register(&command{
		name:  "ping",
		usage: "ping the server",
		run: func(f *frontendModule, sess *session, args []string) error {
			return getPrinter().println("pong").flush(sess)
		},
	})

	// .echo [content]
	register(&command{
		name:   "echo",
		format: "[content]",
		usage:  "echo content",
		run: func(f *frontendModule, sess *session, args []string) error {
			p := getPrinter()
			for i := range args {
				if i > 0 {
					p.print(" ")
				}
				p.print(args[i])
			}
			return p.flush(sess)
		},
	})

	// .send <type> [json]
	register(&command{
		name:   "send",
		format: "<type> [json]",
		usage:  "send message by type with json formatted content",
		run: func(f *frontendModule, sess *session, args []string) error {
			if len(args) < 1 {
				return errorln(sess, "argument <type> required")
			}
			typ, err := proto.ParseType(args[0])
			if err != nil {
				return errorln(sess, "argument <type> invalid")
			}
			body := proto.Text([]byte(strings.Join(args[1:], "")))
			return f.onMessage(sess, proto.Type(typ), body)
		},
	})
}

var (
	crlf = []byte{'\r', '\n'}
	pp   = sync.Pool{
		New: func() interface{} {
			return new(printer)
		},
	}
)

type printer struct {
	err error
	buf bytes.Buffer
}

func getPrinter() *printer {
	p := pp.Get().(*printer)
	p.reset()
	return p
}

func (p *printer) reset() {
	p.err = nil
	p.buf.Reset()
}

func (p *printer) lazyInit() {
	if p.buf.Len() == 0 {
		p.buf.WriteByte(proto.TextResponseType)
	}
}

func errorln(w io.Writer, a ...interface{}) error {
	p := getPrinter()
	p.buf.WriteByte(proto.TextErrorType)
	return p.println(a...).flush(w)
}

func (p *printer) print(a ...interface{}) *printer {
	if p.err == nil {
		p.lazyInit()
		_, p.err = fmt.Fprint(&p.buf, a...)
	}
	return p
}

func (p *printer) println(a ...interface{}) *printer {
	if p.err == nil {
		p.lazyInit()
		_, p.err = fmt.Fprint(&p.buf, a...)
		p.buf.Write(crlf)
	}
	return p
}

func (p *printer) flush(w io.Writer) error {
	if p.err == nil {
		if !bytes.HasSuffix(p.buf.Bytes(), crlf) {
			p.buf.Write(crlf)
		}
		_, p.err = w.Write(p.buf.Bytes())
	}
	err := p.err
	pp.Put(p)
	return err
}
