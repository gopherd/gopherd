package context

import (
	"io/ioutil"

	"google.golang.org/protobuf/compiler/protogen"
)

type File struct {
	GeneratedFile *protogen.GeneratedFile
	Handler       Handler
}

type Handler interface {
	Parse([]byte, error) error
	Output(*protogen.GeneratedFile)
}

type Context struct {
	plugin *protogen.Plugin
	files  map[string]*File
}

func New(plugin *protogen.Plugin) *Context {
	return &Context{
		plugin: plugin,
		files:  make(map[string]*File),
	}
}

func (ctx *Context) Open(filename string, goImportPath protogen.GoImportPath, parser Handler) (*File, error) {
	if file, ok := ctx.files[filename]; ok {
		return file, nil
	}
	file := &File{
		GeneratedFile: ctx.plugin.NewGeneratedFile(filename, goImportPath),
	}
	if parser != nil {
		data, err := ioutil.ReadFile(filename)
		err = parser.Parse(data, err)
		if err != nil {
			return nil, err
		}
		file.Handler = parser
	}
	ctx.files[filename] = file
	return file, nil
}
