package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	log.SetFlags(0)
	rand.Seed(time.Now().UnixNano())

	var (
		cfg      string
		filename string
	)
	flag.StringVar(&cfg, "c", "", "Config filename")
	flag.StringVar(&filename, "e", "", "Script filename")

	flag.BoolVar(&config.Unsafe, "unsafe", false, "Whether run redis in unsafe mode. If not, just readonly commands allowed")
	flag.StringVar(&config.Redis, "redis", "", "Which redis-server to be connected")

	flag.Parse()

	if cfg != "" {
		content, err := ioutil.ReadFile(cfg)
		if err != nil {
			log.Fatalf("read config file %q error: %v", cfg, err)
			os.Exit(1)
		}
		if err := json.Unmarshal(content, &config); err != nil {
			log.Fatalf("parse config file %q error: %v", cfg, err)
			os.Exit(1)
		}
	}
	if filename != "" {
		env := newEnviroment()
		if err := env.init(); err != nil {
			log.Fatalln(err.Error())
			os.Exit(1)
		}
		cmdRun.run(context.Background(), env, []string{filename})
		return
	}

	term := newTerminal()
	if err, ee, ok := toExitError(term.run()); ok {
		os.Exit(ee.code)
	} else if err != nil {
		log.Printf("run terminal error: %v", err)
	}
}
