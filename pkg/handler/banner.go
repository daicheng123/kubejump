package handler

import (
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/utils"
	"io"
	"k8s.io/klog/v2"
	"text/template"
)

type MenuItem struct {
	id       int
	instruct string
	helpText string
}

type Menu []MenuItem

type ColorMeta struct {
	GreenBoldColor string
	ColorEnd       string
}

func (h *InteractiveHandler) displayBanner(sess io.ReadWriter, user string, termConf *entity.TerminalConfig) {
	defaultTitle := utils.WrapperTitle("欢迎登录 KubeJump 开源跳板机系统")
	menu := Menu{
		{id: 1, instruct: "part ClusterName, Namespace, PodIP", helpText: "search login if unique"},
		//{id: 2, instruct: "/ + IP, Hostname, Comment", helpText: "search, such as: /192.168"},
		//{id: 3, instruct: "p", helpText: "display the pod you have permission"},
		//{id: 4, instruct: "g", helpText: "display the node that you have permission"},
		//{id: 5, instruct: "d", helpText: "display the databases that you have permission"},
		//{id: 6, instruct: "k", helpText: "display the kubernetes that you have permission"},
		{id: 2, instruct: "r", helpText: "refresh kubernetes pod assets"},
		//{id: 8, instruct: "s", helpText: "Chinese-English-Japanese switch"},
		{id: 3, instruct: "h", helpText: "print help"},
		{id: 4, instruct: "q", helpText: "exit"},
	}

	prefix := utils.CharClear + utils.CharTab + utils.CharTab
	suffix := utils.CharNewLine + utils.CharNewLine
	welcomeMsg := prefix + utils.WrapperTitle(user+",") + "  " + defaultTitle + suffix
	_, err := io.WriteString(sess, welcomeMsg)
	if err != nil {
		klog.Errorf("Send to client error, %s", err)
		return
	}
	cm := ColorMeta{GreenBoldColor: "\033[1;32m", ColorEnd: "\033[0m"}
	for _, v := range menu {
		line := fmt.Sprintf("\t%d Enter {{.GreenBoldColor}}%s{{.ColorEnd}} to %s.%s",
			v.id, v.instruct, v.helpText, "\r\n")
		tmpl := template.Must(template.New("item").Parse(line))
		if err := tmpl.Execute(sess, cm); err != nil {
			klog.Error(err)
		}
	}
}
