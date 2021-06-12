package main

import (
	"flag"
	"fmt"

	gengo "google.golang.org/protobuf/cmd/protoc-gen-go/internal_gengo"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/gopherd/gopherd/cmd/protoc-gen-gopherd/annotation"
	"github.com/gopherd/gopherd/cmd/protoc-gen-gopherd/context"
)

func main() {
	var (
		flags                 flag.FlagSet
		typeFile              = flags.String("type_file", "", "type filename for store message types")
		typePrefix            = flags.String("type_prefix", "", "type const prefix")
		typeSuffix            = flags.String("type_suffix", "Type", "type const suffix")
		typeMethod            = flags.String("type_method", "Type", "type method name")
		typeRegisty           = flags.String("type_registry", "github.com/gopherd/doge/proto", "typed message registry package")
		typeRegistySizeMethod = flags.String("registry_size_method", "Size", "size method name")
	)
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		var ctx = context.New(gen)
		ctx.Type.Filename = *typeFile
		ctx.Type.Prefix = *typePrefix
		ctx.Type.Suffix = *typeSuffix
		ctx.Type.Method = *typeMethod
		ctx.Type.Registry = *typeRegisty
		ctx.Type.RegistrySizeMethod = *typeRegistySizeMethod
		if ctx.Type.Prefix == "" && ctx.Type.Suffix == "" {
			return fmt.Errorf("gopherd plugin flags type_prefix and type_suffix are both empty")
		}
		for _, f := range gen.Files {
			if f.Generate {
				gengo.GenerateFile(gen, f)
				if err := annotation.Generate(ctx, gen, f); err != nil {
					return err
				}
			}
		}
		gen.SupportedFeatures = gengo.SupportedFeatures
		ctx.Done()
		return nil
	})
}
