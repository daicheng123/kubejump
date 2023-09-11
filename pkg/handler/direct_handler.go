package handler

import (
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/internal/service"
	"github.com/daicheng123/kubejump/pkg/common"
	"github.com/daicheng123/kubejump/pkg/terminal"
	"github.com/daicheng123/kubejump/pkg/utils"
	"github.com/gliderlabs/ssh"
	"io"
	"k8s.io/klog/v2"
	"strconv"
	"time"
	//"golang.org/x/crypto/ssh/terminal"
)

type FormatType int

const (
	FormatNORMAL FormatType = iota
	FormatUUID
	FormatToken
)

type DirectOpt func(*directOpt)

type directOpt struct {
	targetAsset      string
	targetSystemUser string
	User             *entity.User
	terminalConf     *entity.TerminalConfig
	formatType       FormatType

	tokenInfo *entity.ConnectTokenInfo
	//sftpMode bool
}

func (d directOpt) IsTokenConnection() bool {
	return d.formatType == FormatToken
}

func DirectTargetAsset(targetAsset string) DirectOpt {
	return func(opt *directOpt) {
		opt.targetAsset = targetAsset
	}
}

func DirectTargetSystemUser(targetSystemUser string) DirectOpt {
	return func(opts *directOpt) {
		opts.targetSystemUser = targetSystemUser
	}
}

func DirectUser(User *entity.User) DirectOpt {
	return func(opts *directOpt) {
		opts.User = User
	}
}

func DirectTerminalConf(conf *entity.TerminalConfig) DirectOpt {
	return func(opts *directOpt) {
		opts.terminalConf = conf
	}
}

func DirectFormatType(format FormatType) DirectOpt {
	return func(opts *directOpt) {
		opts.formatType = format
	}
}

func DirectConnectToken(tokenInfo *entity.ConnectTokenInfo) DirectOpt {
	return func(opts *directOpt) {
		opts.tokenInfo = tokenInfo
	}
}

type DirectHandler struct {
	term        *terminal.Terminal
	sess        ssh.Session
	wrapperSess *WrapperSession
	opts        *directOpt
	jmsService  *service.JMService

	assets []*entity.Asset

	selectedSystemUser *entity.User

	i18nLang string
}

func NewDirectHandler(session ssh.Session, jmsService *service.JMService, optSetters ...DirectOpt) (*DirectHandler, error) {
	opts := &directOpt{}
	for i := range optSetters {
		optSetters[i](opts)
	}
	var (
		selectedAssets []*entity.Asset
		err            error
		wrapperSess    *WrapperSession
		term           *terminal.Terminal
		errMsg         string
	)

	defer func() {
		if err != nil {
			utils.IgnoreErrWriteString(session, errMsg)
		}
	}()

	if !opts.IsTokenConnection() {
		selectedAssets, err = jmsService.ListPodAsset(session.Context(), opts.targetAsset)
		if err != nil {
			klog.Errorf("Get direct asset failed: %s", err)
			errMsg = "Core API failed"
			return nil, err
		}

		if len(selectedAssets) <= 0 {
			msg := fmt.Sprintf("not found matched asset %s", opts.targetAsset)
			errMsg = msg + "\r\n"
			err = fmt.Errorf("no found matched asset: %s", opts.targetAsset)
			return nil, err
		}
	}

	wrapperSess = NewWrapperSession(session)
	term = terminal.NewTerminal(wrapperSess, "Opt> ")

	d := &DirectHandler{
		opts:       opts,
		sess:       session,
		jmsService: jmsService,
		assets:     selectedAssets,
		//i18nLang:   i18nLang,
		wrapperSess: wrapperSess,
		term:        term,
	}
	return d, nil
}

func (d *DirectHandler) WatchWinSizeChange(winChan <-chan ssh.Window) {
	defer klog.Infof("Request %s: Windows change watch close", d.wrapperSess.Uuid)
	for {
		select {
		case <-d.sess.Context().Done():
			return
		case win, ok := <-winChan:
			if !ok {
				return
			}
			d.wrapperSess.SetWin(win)
			klog.Infof("Term window size change: %d*%d", win.Height, win.Width)
			_ = d.term.SetSize(win.Width, win.Height)
		}
	}
}

func (d *DirectHandler) Dispatch() {
	_, winChan, _ := d.sess.Pty()
	go d.WatchWinSizeChange(winChan)
	if d.opts.IsTokenConnection() {
		//d.LoginConnectToken()
		return
	}
	d.LoginAsset()
}

func (d *DirectHandler) LoginAsset() {
	checkChan := make(chan bool)
	//d.jmsService.ListPodAsset(context.Background(), "")
	go d.checkMaxIdleTime(checkChan)

	term := d.term

	idLabel := "ID"
	clusterLabel := "Cluster"
	namespaceLabel := "Namespace"
	podLabel := "PodName"
	podIPLabel := "PodIP"
	podStatusLabel := "PodStatus"

	labels := []string{idLabel, clusterLabel, namespaceLabel, podLabel, podIPLabel, podStatusLabel}
	fields := []string{"ID", "Cluster", "Namespace", "PodName", "PodIP", "PodStatus"}
	data := make([]map[string]string, len(d.assets))

	for i := range d.assets {
		row := make(map[string]string)
		row["ID"] = strconv.Itoa(i + 1)
		row["Cluster"] = d.assets[i].ClusterName
		row["Namespace"] = d.assets[i].Namespace
		row["PodName"] = d.assets[i].PodName
		row["PodIP"] = d.assets[i].PodIP
		row["Status"] = d.assets[i].PodStatus
		data[i] = row
	}

	w, _ := d.term.GetSize()

	table := common.WrapperTable{
		Fields: fields,
		Labels: labels,
		FieldsSize: map[string][3]int{
			"ID":          {0, 0, 5},
			"ClusterName": {0, 40, 0},
			"Namespace":   {0, 15, 40},
			"PodName":     {0, 0, 0},
		},
		Data:        data,
		TotalSize:   w,
		TruncPolicy: 2,
	}
	table.Initial()

	loginTip := "select one pod to login"

	_, _ = term.Write([]byte(utils.CharClear))
	_, _ = term.Write([]byte(table.Display()))

	utils.IgnoreErrWriteString(term, utils.WrapperString(loginTip, utils.Green))
	utils.IgnoreErrWriteString(term, utils.CharNewLine)
	utils.IgnoreErrWriteString(term, utils.WrapperString(d.opts.targetAsset, utils.Green))
	utils.IgnoreErrWriteString(term, utils.CharNewLine)
}

func (d *DirectHandler) displayAssets(assets []entity.Asset) {

}

func (d *DirectHandler) checkMaxIdleTime(checkChan chan bool) {
	maxIdleMinutes := d.opts.terminalConf.MaxIdleTime
	checkMaxIdleTime(maxIdleMinutes, d.opts.User,
		d.sess, checkChan)
}

func checkMaxIdleTime(maxIdleMinutes int, user *entity.User, sess ssh.Session, checkChan <-chan bool) {
	maxIdleTime := time.Duration(maxIdleMinutes) * time.Minute
	tick := time.NewTicker(maxIdleTime)
	defer tick.Stop()
	checkStatus := true
	for {
		select {
		case <-tick.C:
			if checkStatus {
				msg := fmt.Sprintf("Connect idle more than %d minutes, disconnect", maxIdleMinutes)
				_, _ = io.WriteString(sess, "\r\n"+msg+"\r\n")
				_ = sess.Close()
				klog.Infof("User %s input idle more than %d minutes", user.Name, maxIdleMinutes)
			}
		case <-sess.Context().Done():
			klog.Infof("Stop checking user %s input idle time", user.Name)
			return
		case checkStatus = <-checkChan:
			if !checkStatus {
				klog.Infof("Stop checking user %s idle time if more than %d minutes", user.Name, maxIdleMinutes)
				continue
			}
			tick.Reset(maxIdleTime)
			klog.Infof("Start checking user %s idle time if more than %d minutes", user.Name, maxIdleMinutes)
		}
	}
}
