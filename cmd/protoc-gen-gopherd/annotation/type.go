package annotation

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/gopherd/doge/encoding"
	"google.golang.org/protobuf/compiler/protogen"
)

// @Type
// @Type(value: int32)
// @Type(source: string[, min=int32, max=int32])
type Type struct {
	Oneof struct {
		Empty  bool
		Value  int32
		Source struct {
			Filename string
			Min, Max int32
		}
	}
}

func (*Type) Name() string { return AnnotationType }

func (t *Type) VerifyDuplicate(associated associated, _ Annotation) error {
	return t.errorf(associated, "@Type duplicated")
}

func (t *Type) valid(associated associated) error {
	switch {
	case associated.oneof.pkg:
		if t.Oneof.Source.Filename == "" {
			return t.errorf(associated, "@Type first argument must be a string for package")
		}
		if t.Oneof.Source.Max < t.Oneof.Source.Min {
			return t.errorf(associated, "@Type max (%d) less than min (%d)", t.Oneof.Source.Max, t.Oneof.Source.Min)
		}
	case associated.oneof.message != nil:
		if t.Oneof.Source.Filename != "" {
			return t.errorf(associated, "@Type should has no arguments or has an integer argument for message")
		}
	default:
		return t.errorf(associated, "@Type only supported for package and message")
	}
	return nil
}

func (t *Type) errorf(associated associated, format string, args ...interface{}) error {
	prefix := fmt.Sprintf("%s:%d: ", associated.filename(), associated.lineno)
	return errors.New(prefix + fmt.Sprintf(format, args...))
}

func parseTypeAnnotation(associated associated, parser *encoding.Parser) (ann Annotation, err error) {
	if err = parser.Next(); err != nil {
		return
	}
	m := &Type{}
	defer func() {
		if err == nil {
			err = m.valid(associated)
		}
	}()

	if parser.Tok == scanner.EOF {
		m.Oneof.Empty = true
		ann = m
		return
	}
	if err = parser.Expect('('); err != nil {
		return
	}
	switch parser.Tok {
	case scanner.String:
		m.Oneof.Source.Filename = parser.Lit
		if err = parser.Next(); err != nil {
			return
		}
		// parse kvpairs
		seen := make(map[string]bool)
		for parser.Tok == ',' {
			if err = parser.Next(); err != nil {
				return
			}
			key := parser.Lit
			if err = parser.Expect(scanner.Ident); err != nil {
				return
			}
			if err = parser.Expect('='); err != nil {
				return
			}
			value := parser.Lit
			if err = parser.Expect(scanner.Int); err != nil {
				return
			}
			var x int
			x, err = strconv.Atoi(value)
			if err != nil {
				return
			}
			if x < math.MinInt32 || x > math.MaxInt32 {
				err = fmt.Errorf("%s %d out of range [%d, %d]", AnnotationType, x, math.MinInt32, math.MaxInt32)
				return
			}
			if seen[key] {
				err = fmt.Errorf("%s named argument %s duplicated", AnnotationType, key)
				return
			}
			switch key {
			case "min":
				m.Oneof.Source.Min = int32(x)
			case "max":
				m.Oneof.Source.Max = int32(x)
			default:
				err = fmt.Errorf("%s named argument %s unrecognized", AnnotationType, key)
				return
			}
		}
	case scanner.Int:
		var value int64
		value, err = strconv.ParseInt(parser.Lit, 0, 32)
		if err != nil {
			return
		}
		if value > math.MaxInt32 || value < math.MinInt32 {
			err = fmt.Errorf("%s %d out of range [%d, %d]", AnnotationType, value, math.MinInt32, math.MaxInt32)
			return
		}
		m.Oneof.Value = int32(value)
		if err = parser.Next(); err != nil {
			return
		}
	default:
		err = parser.ExpectError(scanner.String, scanner.Int)
		return
	}
	if err = parser.Expect(')'); err != nil {
		return
	}
	if err = parser.Expect(scanner.EOF); err != nil {
		return
	}

	ann = m
	return
}

func generateTypeAnnotation(gen *protogen.Plugin, f *protogen.File, g *protogen.GeneratedFile, anns []*associatedAnnotation) error {
	var (
		constCnt   int
		messageCnt int
	)
	for _, ann := range anns {
		if ann.associated.oneof.message == nil {
			continue
		}
		messageType := ann.annotation.(*Type)
		if messageType.Oneof.Source.Filename != "" {
			continue
		}
		messageCnt++
		if !messageType.Oneof.Empty {
			constCnt++
		}
	}
	if constCnt > 0 {
		g.P()
		g.P("const (")
		for _, ann := range anns {
			if ann.associated.oneof.message == nil {
				continue
			}
			messageType := ann.annotation.(*Type)
			if messageType.Oneof.Empty {
				continue
			}
			if messageType.Oneof.Source.Filename != "" {
				continue
			}
			g.P("\t", ann.associated.oneof.message.GoIdent.GoName, "Type = ", messageType.Oneof.Value)
		}
		g.P(")")
	}

	if messageCnt > 0 {
		g.P()
		for _, ann := range anns {
			if ann.associated.oneof.message == nil {
				continue
			}
			messageType := ann.annotation.(*Type)
			if messageType.Oneof.Source.Filename != "" {
				continue
			}
			name := ann.associated.oneof.message.GoIdent.GoName
			g.P("\tfunc (*", name, ") Type() int32 { return ", name, "Type }")
		}
	}

	return nil
}

type _type struct {
	name  string
	value int32
}

type typeGenerator struct {
	min, max  int32
	types     []_type
	name2type map[string]int32
	type2name map[int32]string
}

func newTypeGenerator(min, max int32) *typeGenerator {
	return &typeGenerator{
		min:       min,
		max:       max,
		name2type: make(map[string]int32),
		type2name: make(map[int32]string),
	}
}

func (gen *typeGenerator) sort() {
	sort.Slice(gen.types, func(i, j int) bool {
		return gen.types[i].value < gen.types[j].value
	})
}

func (gen *typeGenerator) add(name string, t Type) error {
	if _, ok := gen.name2type[name]; ok {
		return nil
	}
	var value int32
	if t.Oneof.Empty {
		// TODO
	} else {
		value = t.Oneof.Value
	}
	gen.name2type[name] = value
	return nil
}

// parse types.proto
//
//	...
// enum <Name> {
// 	...
// }
// 	...
func parse_types_proto(data []byte, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}
	return nil, nil
}

type typesProtoContent struct {
	leading  string
	enumName string
	trailing string

	generator *typeGenerator
}

func newTypesProtoContent(min, max int32) *typesProtoContent {
	return &typesProtoContent{
		generator: newTypeGenerator(min, max),
	}
}

var typesProtoRegexp = regexp.MustCompile(`(?ms)(.*)enum[[:space:]]?([a-zA-Z_]?[a-zA-Z0-9_]*)[[:space:]]?{(.*)}(.*)`)

func (content *typesProtoContent) Parse(data []byte, err error) error {
	if err != nil {
		return err
	}
	matched := typesProtoRegexp.FindAllSubmatch(data, -1)
	if len(matched) == 0 {
		return fmt.Errorf("types.proto")
	}
	return nil
}

func (content *typesProtoContent) Output(g *protogen.GeneratedFile) {
	var (
		maxNameLength int
		missingZero   = true
	)
	for i, t := range content.generator.types {
		if t.value == 0 {
			missingZero = false
		}
		if i == 0 || len(t.name) > maxNameLength {
			maxNameLength = len(t.name)
		}
	}
	if missingZero {
		content.generator.types = append(content.generator.types, _type{
			name:  "_UnusedType",
			value: 0,
		})
	}
	content.generator.sort()
	g.P(content.leading)
	g.P("enum ", content.enumName, " {")
	for _, t := range content.generator.types {
		g.P("\t", t.name, strings.Repeat(" ", maxNameLength-len(t.name)), " = ", t.value, ",")
	}
	g.P("}")
	g.P(content.trailing)
}
