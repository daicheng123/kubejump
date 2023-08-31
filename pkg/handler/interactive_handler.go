package handler

import (
	"context"
	"fmt"
	"github.com/daicheng123/kubejump/config"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/internal/service"
	"github.com/daicheng123/kubejump/pkg/terminal"
	"github.com/gliderlabs/ssh"
	"k8s.io/klog/v2"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	PAGESIZEALL = 0
)

func NewInteractiveHandler(sess ssh.Session, user *entity.User, jmsService *service.JMService) *InteractiveHandler {
	wrapperSess := NewWrapperSession(sess)
	term := terminal.NewTerminal(wrapperSess, "Opt> ")
	handler := &InteractiveHandler{
		sess:       wrapperSess,
		user:       user,
		term:       term,
		jmsService: jmsService,
	}

	handler.Initial()
	return handler
}

type InteractiveHandler struct {
	sess *WrapperSession
	user *entity.User
	term *terminal.Terminal

	wg sync.WaitGroup

	jmsService *service.JMService

	terminalConf *entity.TerminalConfig

	selectHandler   *UserSelectHandler
	nodes           []entity.Asset
	assetLoadPolicy string
}

func (h *InteractiveHandler) Initial() {
	conf := config.GetConf()
	if conf.ClientAliveInterval > 0 {
		go h.keepSessionAlive(time.Duration(conf.ClientAliveInterval) * time.Second)
	}
	h.assetLoadPolicy = strings.ToLower(conf.AssetLoadPolicy)
	h.displayHelp()

	h.selectHandler = &UserSelectHandler{
		user:     h.user,
		h:        h,
		pageInfo: &pageInfo{},
	}

	switch h.assetLoadPolicy {
	case "all":
		allAssets, err := h.jmsService.ListPodAsset(h.sess.Context(), "")
		if err != nil {
			klog.Errorf("Get all user perms assets failed: %s", err)
		}
		h.selectHandler.SetAllLocalAssetData(allAssets)
	}

	h.firstLoadData()
}

func (h *InteractiveHandler) displayHelp() {
	h.term.SetPrompt("Opt> ")
	h.displayBanner(h.sess, h.user.Name, h.terminalConf)
}

func (h *InteractiveHandler) keepSessionAlive(keepAliveTime time.Duration) {
	t := time.NewTicker(keepAliveTime)
	defer t.Stop()
	for {
		select {
		case <-h.sess.Sess.Context().Done():
			return
		case <-t.C:
			_, err := h.sess.Sess.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				klog.Errorf("Request %s: Send user %s keepalive packet failed: %s",
					h.sess.Uuid, h.user.Name, err)
				continue
			}
			klog.Infof("Request %s: Send user %s keepalive packet success", h.sess.Uuid, h.user.Name)
		}
	}
}

func (h *InteractiveHandler) firstLoadData() {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.loadUserPodAssets()
	}()
}

func (h *InteractiveHandler) loadUserPodAssets() {
	nodes, err := h.jmsService.ListPodAsset(context.Background(), "")
	if err != nil {
		klog.Errorf("Get user nodes error: %s", err)
		return
	}
	h.nodes = nodes
}

func (h *InteractiveHandler) WatchWinSizeChange(winChan <-chan ssh.Window) {
	defer klog.Infof("Request %s: Windows change watch close", h.sess.Uuid)
	for {
		select {
		case <-h.sess.Sess.Context().Done():
			return
		case win, ok := <-winChan:
			if !ok {
				return
			}
			h.sess.SetWin(win)
			klog.Infof("Term window size change: %d*%d", win.Height, win.Width)
			_ = h.term.SetSize(win.Width, win.Height)
		}
	}
}

func (h *InteractiveHandler) Dispatch() {
	defer klog.Infof("Request %s: User %s stop interactive", h.sess.ID(), h.user.Name)
	var initialed bool
	checkChan := make(chan bool)
	go h.checkMaxIdleTime(checkChan)
	for {
		checkChan <- true
		line, err := h.term.ReadLine()
		if err != nil {
			klog.Infof("User %s close connect %s", h.user.Name, err)
			break
		}
		checkChan <- false
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			// 当 只是回车 空字符单独处理
			if initialed {
				fmt.Printf("hello\n")
				h.selectHandler.MoveNextPage()
			} else {
				fmt.Printf("world\n")
				h.selectHandler.SetSelectType(TypeAsset)
				h.selectHandler.Search("")
			}
			initialed = true
			continue
		}
		initialed = true
		switch len(line) {
		case 1:
			switch strings.ToLower(line) {
			case "h":
				h.displayHelp()
				initialed = false
				continue
			case "q":
				klog.Infof("user %s enter %s to exit", h.user.Name, line)
				return
			case "r":
				klog.Infof("user %s enter %s to exit", h.user.Name, line)
				return
			}
		default:
			switch {
			case line == "exit", line == "quit":
				klog.Infof("user %s enter %s to exit", h.user.Name, line)
				return
			}
		}
		h.selectHandler.SearchOrProxy(line)
	}
}

func (h *InteractiveHandler) checkMaxIdleTime(checkChan <-chan bool) {
	//maxIdleMinutes := h.terminalConf.MaxIdleTime
	maxIdleMinutes := config.GetConf().TerminalConf.MaxIdleTime
	checkMaxIdleTime(maxIdleMinutes, h.user, h.sess.Sess, checkChan)
}

func getPageSize(term *terminal.Terminal, termConf *entity.TerminalConfig) int {
	var (
		pageSize  int
		minHeight = 8 // 分页显示的最小高度

	)
	_, height := term.GetSize()

	klog.Infoln("[getPageSize] AssetListPageSize is %s", termConf.AssetListPageSize)
	AssetListPageSize := termConf.AssetListPageSize
	switch AssetListPageSize {
	case "auto":
		pageSize = height - minHeight
	case "all":
		return PAGESIZEALL
	default:
		if value, err := strconv.Atoi(AssetListPageSize); err == nil {
			pageSize = value
		} else {
			pageSize = height - minHeight
		}
	}
	fmt.Printf("height: %d, minHeight: %d, pageSize: %d\n", height, minHeight, pageSize)
	if pageSize <= 0 {
		pageSize = 1
	}
	return pageSize
}
