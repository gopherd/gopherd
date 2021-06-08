package main

import (
	"flag"

	gengo "google.golang.org/protobuf/cmd/protoc-gen-go/internal_gengo"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/gopherd/gopherd/cmd/protoc-gen-gopherd/annotation"
)

func main() {
	var flags flag.FlagSet
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if f.Generate {
				if err := annotation.Generate(gen, f); err != nil {
					return err
				}
			}
		}
		gen.SupportedFeatures = gengo.SupportedFeatures
		return nil
	})
}
