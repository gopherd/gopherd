package frontendmod

import (
	"bytes"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/text/resp"
)

type command struct {
	name   string
	format string
	usage  string
	run    func(*frontendModule, *session, *resp.Command) error
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
	// command [command]
	register(&command{
		name:   "command",
		format: "[commands...]",
		usage:  "show commands help information",
		run: func(mod *frontendModule, sess *session, cmd *resp.Command) error {
			var (
				cmds []*command
			)
			if n := cmd.NArg(); n > 0 {
				for i := 0; i < n; i++ {
					name := cmd.Arg(i)
					c := commands[strings.ToLower(name)]
					if c == nil {
						return errorln(sess, "command", name, "not found")
					}
					cmds = append(cmds, c)
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
			p.setType(resp.ArrayType.Byte())
			p.println(strconv.Itoa(len(cmds)))
			alignSize := 0
			prefix := string(resp.StringType.Byte())
			for _, c := range cmds {
				var size int
				if c.format != "" {
					size = len(prefix) + len(c.name) + len(c.format) + 1
				} else {
					size = len(prefix) + len(c.name)
				}
				if size > alignSize {
					alignSize = size
				}
			}

			for _, c := range cmds {
				var left string
				if c.format != "" {
					left = prefix + c.name + " " + c.format
				} else {
					left = prefix + c.name
				}
				p.println(left + strings.Repeat(" ", alignSize-len(left)+4) + c.usage)
			}
			return p.flush(sess)
		},
	})

	// ping [content]
	register(&command{
		name:  "ping",
		usage: "ping the server",
		run: func(mod *frontendModule, sess *session, cmd *resp.Command) error {
			return getPrinter().println("pong").flush(sess)
		},
	})

	// echo [content]
	register(&command{
		name:   "echo",
		format: "[content]",
		usage:  "echo content",
		run: func(mod *frontendModule, sess *session, cmd *resp.Command) error {
			p := getPrinter()
			for i, n := 0, cmd.NArg(); i < n; i++ {
				if i > 0 {
					p.print(" ")
				}
				p.print(cmd.Arg(i))
			}
			return p.flush(sess)
		},
	})

	// send <type> [json]
	register(&command{
		name:   "send",
		format: "<type> [json]",
		usage:  "send message by type with json formatted content",
		run: func(mod *frontendModule, sess *session, cmd *resp.Command) error {
			argc := cmd.NArg()
			if argc < 1 {
				return errorln(sess, "argument <type> required")
			}
			typ, err := proto.ParseType(cmd.Arg(0))
			if err != nil {
				return errorln(sess, "argument <type> invalid")
			}
			switch argc {
			case 1:
				return mod.onMessage(sess, typ, proto.Text(nil))
			case 2:
				return mod.onMessage(sess, typ, proto.Text([]byte(cmd.Arg(1))))
			default:
				return resp.ErrNumberOfArguments
			}
		},
	})
}

var (
	crlf = []byte{'\r', '\n'}
	pp   = sync.Pool{
		New: func() any {
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
		p.buf.WriteByte(resp.StringType.Byte())
	}
}

func errorln(w io.Writer, a ...string) error {
	p := getPrinter()
	p.buf.WriteByte(resp.ErrorType.Byte())
	return p.println(a...).flush(w)
}

func (p *printer) setType(b byte) {
	if p.buf.Len() == 0 {
		p.buf.WriteByte(b)
	} else {
		p.buf.Bytes()[0] = b
	}
}

func (p *printer) print(a ...string) *printer {
	if p.err == nil {
		p.lazyInit()
		for i := range a {
			if i > 0 {
				p.buf.WriteByte(' ')
			}
			p.buf.WriteString(a[i])
		}
	}
	return p
}

func (p *printer) println(a ...string) *printer {
	if p.err == nil {
		p.print(a...)
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
