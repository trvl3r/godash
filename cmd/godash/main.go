package main

import (
	"fmt"
	"net/http"
	"os"

	flags "github.com/jessevdk/go-flags"

	"godash/internal/configs"
)
func main() {
	var options struct {
		Config      string `short:"c" long:"config" description:"Where's the config file place, default ./internal/configs/config.yaml"`
		Environment string `short:"e" long:"environment" default:"development"`
	}

	p := flags.NewParser(&options, flags.Default)
	if _, err := p.Parse(); err != nil {
		log.Panicln(err)
	}

	if options.Config == "" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Panicln(err)
		}
		back := ".."
		if strings.Contains(dir, "cmd") {
			back = "../.."
		}
		options.Config = path.Join(dir, back, "internal/configs/config.yaml")
	}

	if err := configs.Init(options.Config, options.Environment); err != nil {
		log.Panicln(err)
	}

	config := configs.AppConfig

	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	fmt.Println("working dir is: ", path)
	http_dir := "../app/public/"


	if err != nil {
		fmt.Println(err)
	}


	if err := startHTTP(db, logger, config.HTTP.Port); err != nil {
		log.Panicln(err)
	}

}
func StartHTTP(dir, port) {
	fs := http.FileServer(http.Dir(dir))
	http.Handle("/", fs)
	port := string(":", port)
	http.ListenAndServe(port, nil)
}
