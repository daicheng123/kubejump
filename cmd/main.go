package main

import (
	"flag"
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
	//var stopChan = make(chan struct{})
	//app.RunForever(cfgFile)

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
