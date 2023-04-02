package main

import (
	"flag"
	"github.com/daicheng123/kubejump/cmd/app"
)

var (
	cfgFile  string
	logLevel int
)

func init() {
	flag.StringVar(&cfgFile, "f", "config.yml", "config.yml path")
	flag.IntVar(&logLevel, "l", 1, "loglevel number")
}

func main() {
	flag.Parse()
	app.RunForever(cfgFile)
}
