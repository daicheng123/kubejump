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
	flag.StringVar(&cfgFile, "config", "config.yml", "--config [config.yml path]")
	flag.IntVar(&logLevel, "level", 1, "--level [loglevel number] ")
}

func main() {
	flag.Parse()
	//var stopChan = make(chan struct{})
	app.RunForever(cfgFile)

	//go wait.NonSlidingUntil(func() {
	//	fmt.Println("hello")
	//}, time.Second, stopChan)
	//wait.Until(func() {
	//	fmt.Println("world")
	//}, time.Second, stopChan)

	//labels.Parse("app=nginx")

	//fmt.Println(labels.FormatLabels(map[string]string{"app": "nginx"}))
	//fmt.Println(labels.Parse(labels.FormatLabels(map[string]string{"app": "nginx"})))
	//labels.SelectorFromSet()

	//fmt.Println())

}
