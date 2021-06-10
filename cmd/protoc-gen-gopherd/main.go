package main

import (
	"flag"

	gengo "google.golang.org/protobuf/cmd/protoc-gen-go/internal_gengo"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/gopherd/gopherd/cmd/protoc-gen-gopherd/annotation"
	"github.com/gopherd/gopherd/cmd/protoc-gen-gopherd/context"
)

func main() {
	var (
		flags     flag.FlagSet
		typesFile = flags.String("types_file", "", "types filename for store message types")
	)
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		var ctx = context.New(gen)
		ctx.TypesFilename = *typesFile
		for _, f := range gen.Files {
			if f.Generate {
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
