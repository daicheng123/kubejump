package handler

import (
	"context"
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
	h.displayHelp()

	h.selectHandler = &UserSelectHandler{
		user:     h.user,
		h:        h,
		pageInfo: &pageInfo{},
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

	pageSize := getPageSize(h.term, config.GetConf().TerminalConf)
	resp, err := h.jmsService.ListPodsFromStorage(context.Background(), &entity.PaginationParam{
		PageSize: pageSize,
		IsActive: true,
		SortBy:   "cluster_ref desc",
		Offset:   0,
	})
	if err != nil {
		klog.Errorf("Get user nodes error: %s", err)
		return
	}
	h.selectHandler.SetAllLocalAssetData(resp.Data)
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
				h.selectHandler.MoveNextPage()
			} else {
				h.selectHandler.SetSelectPrepare()
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
			case "b":
				h.selectHandler.MovePrePage()
				continue
			case "n":
				h.selectHandler.MoveNextPage()
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

			case strings.Index(line, "/") == 0:
				if strings.Index(line[1:], "/") == 0 {
					line = strings.TrimSpace(line[2:])
					h.selectHandler.Search(line)
					continue
				}
				line = strings.TrimSpace(line[1:])
				h.selectHandler.Search(line)
				continue
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
	AssetListPageSize := config.GlobalConfig.TerminalConf.AssetListPageSize
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
	if pageSize <= 0 {
		pageSize = 1
	}
	return pageSize
}

func (u *UserSelectHandler) MovePrePage() {
	if u.HasPrev() {
		offset := u.CurrentOffSet()
		newPageSize := getPageSize(u.h.term, u.h.terminalConf)
		start := offset - newPageSize*2
		if start <= 0 {
			start = 0
		}
		u.currentResult = u.Retrieve(newPageSize, start, u.searchKey)
	}
	u.DisplayCurrentResult()
}
