package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/daicheng123/kubejump/pkg/exchange"
	"github.com/daicheng123/kubejump/pkg/srvconn"
	"github.com/daicheng123/kubejump/pkg/utils"
	"k8s.io/klog/v2"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

type SwitchSession struct {
	ID string

	MaxIdleTime   int
	keepAliveTime int

	ctx    context.Context
	cancel context.CancelFunc

	proxy *ProxyServer

	currentOperator atomic.Value // 终断会话的管理员名称

	pausedStatus atomic.Bool // 暂停状态

	notifyMsgChan chan *exchange.RoomMessage

	MaxSessionTime time.Time
}

func (s *SwitchSession) Terminate(username string) {
	select {
	case <-s.ctx.Done():
		return
	default:
		s.setOperator(username)
	}
	s.cancel()
	klog.Infof("Session[%s] receive terminate task from %s", s.ID, username)
}

func (s *SwitchSession) setOperator(username string) {
	s.currentOperator.Store(username)
}

func (s *SwitchSession) PauseOperation(username string) {
	s.pausedStatus.Store(true)
	s.setOperator(username)
	klog.Infof("Session[%s] receive pause task from %s", s.ID, username)
	p, _ := json.Marshal(map[string]string{"user": username})
	s.notifyMsgChan <- &exchange.RoomMessage{
		Event: exchange.PauseEvent,
		Body:  p,
	}
}

func (s *SwitchSession) ResumeOperation(username string) {
	s.pausedStatus.Store(false)
	s.setOperator(username)
	klog.Infof("Session[%s] receive resume task from %s", s.ID, username)
	p, _ := json.Marshal(map[string]string{"user": username})
	s.notifyMsgChan <- &exchange.RoomMessage{
		Event: exchange.ResumeEvent,
		Body:  p,
	}
}

// Bridge 桥接两个链接
func (s *SwitchSession) Bridge(userConn UserConnection, srvConn srvconn.ServerConnection) (err error) {

	//parser := s.proxy.GetFilterParser()
	//klog.Infof("Conn[%s] create ParseEngine success", userConn.ID())
	//replayRecorder := s.proxy.GetReplayRecorder()
	//klog.Infof("Conn[%s] create replay success", userConn.ID())
	srvInChan := make(chan []byte, 1)
	done := make(chan struct{})
	userInputMessageChan := make(chan *exchange.RoomMessage, 1)
	// 处理数据流
	userOutChan, srvOutChan := ParseStream(userInputMessageChan, srvInChan, done)

	defer func() {
		//close(done)
		_ = userConn.Close()
		_ = srvConn.Close()
		done <- struct{}{}
	}()

	winCh := userConn.WinCh()
	maxIdleTime := time.Duration(s.MaxIdleTime) * time.Minute
	lastActiveTime := time.Now()
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()

	room := exchange.CreateRoom(s.ID, userInputMessageChan)
	exchange.Register(room)
	defer exchange.UnRegister(room)
	conn := exchange.WrapperUserCon(userConn)
	room.Subscribe(conn)
	defer room.UnSubscribe(conn)
	exitSignal := make(chan struct{}, 2)
	go func() {
		var (
			exitFlag bool
		)
		buffer := bytes.NewBuffer(make([]byte, 0, 1024*2))
		/*
		 这里使用了一个buffer，将用户输入的数据进行了分包，分包的依据是utf8编码的字符。
		*/
		maxLen := 1024
		for {
			buf := make([]byte, maxLen)
			nr, err2 := srvConn.Read(buf)
			validBytes := buf[:nr]
			if nr > 0 {
				bufferLen := buffer.Len()
				if bufferLen > 0 || nr == maxLen {
					buffer.Write(buf[:nr])
					validBytes = validBytes[:0]
				}
				remainBytes := buffer.Bytes()
				for len(remainBytes) > 0 {
					r, size := utf8.DecodeRune(remainBytes)
					if r == utf8.RuneError {
						// utf8 max 4 bytes
						if len(remainBytes) <= 3 {
							break
						}
					}
					validBytes = append(validBytes, remainBytes[:size]...)
					remainBytes = remainBytes[size:]
				}
				buffer.Reset()
				if len(remainBytes) > 0 {
					buffer.Write(remainBytes)
				}
				select {
				case srvInChan <- validBytes:
				case <-done:
					exitFlag = true
					klog.Infof("Session[%s] done", s.ID)
				}
				if exitFlag {
					break
				}
			}
			if err2 != nil {
				klog.Errorf("Session[%s] srv read err: %s", s.ID, err2)
				break
			}
		}
		klog.Infof("Session[%s] srv read end", s.ID)
		exitSignal <- struct{}{}
		close(srvInChan)
	}()
	user := s.proxy.connOpts.authInfo.User
	meta := exchange.MetaMessage{
		UserId:     user.Name,
		User:       user.String(),
		Created:    utils.NewNowUTCTime().String(),
		RemoteAddr: userConn.RemoteAddr(),
		TerminalId: userConn.ID(),
		Primary:    true,
		Writable:   true,
	}
	room.Broadcast(&exchange.RoomMessage{
		Event: exchange.ShareJoin,
		Meta:  meta,
	})
	//if parser.zmodemParser != nil {
	//	parser.zmodemParser.FireStatusEvent = func(event zmodem.StatusEvent) {
	//		msg := exchange.RoomMessage{Event: exchange.ActionEvent}
	//		switch event {
	//		case zmodem.StartEvent:
	//			msg.Body = []byte(exchange.ZmodemStartEvent)
	//		case zmodem.EndEvent:
	//			msg.Body = []byte(exchange.ZmodemEndEvent)
	//		default:
	//			msg.Body = []byte(event)
	//		}
	//		room.Broadcast(&msg)
	//	}
	//}
	go func() {
		for {
			buf := make([]byte, 1024)
			nr, err := userConn.Read(buf)
			if nr > 0 {
				index := bytes.IndexFunc(buf[:nr], func(r rune) bool {
					return r == '\r'
				})
				//if index <= 0 || !parser.NeedRecord() {
				//	room.Receive(&exchange.RoomMessage{
				//		Event: exchange.DataEvent, Body: buf[:nr],
				//		Meta: meta})
				//} else {
				room.Receive(&exchange.RoomMessage{
					Event: exchange.DataEvent, Body: buf[:index],
					Meta: meta})
				time.Sleep(time.Millisecond * 100)
				room.Receive(&exchange.RoomMessage{
					Event: exchange.DataEvent, Body: buf[index:nr],
					Meta: meta})
				//}
			}
			if err != nil {
				klog.Errorf("Session[%s] user read err: %s", s.ID, err)
				break
			}
		}
		klog.Infof("Session[%s] user read end", s.ID)
		exitSignal <- struct{}{}
	}()
	keepAliveTime := time.Duration(s.keepAliveTime) * time.Second
	keepAliveTick := time.NewTicker(keepAliveTime)
	defer keepAliveTick.Stop()
	//lang := s.proxy.connOpts.getLang()
	for {
		select {
		// 检测是否超过最大空闲时间
		case now := <-tick.C:
			if s.MaxSessionTime.Before(now) {
				msg := "Session max time reached, disconnect"
				klog.Infof("Session[%s] max session time reached, disconnect", s.ID)
				msg = utils.WrapperWarn(msg)
				//replayRecorder.Record([]byte(msg))
				room.Broadcast(&exchange.RoomMessage{Event: exchange.DataEvent, Body: []byte("\n\r" + msg)})
				return
			}

			outTime := lastActiveTime.Add(maxIdleTime)
			if now.After(outTime) {
				msg := fmt.Sprintf("Connect idle more than %d minutes, disconnect", s.MaxIdleTime)
				klog.Infof("Session[%s] idle more than %d minutes, disconnect", s.ID, s.MaxIdleTime)
				msg = utils.WrapperWarn(msg)
				//replayRecorder.Record([]byte(msg))
				room.Broadcast(&exchange.RoomMessage{Event: exchange.DataEvent, Body: []byte("\n\r" + msg)})
				return
			}
			if s.proxy.CheckPermissionExpired(now) {
				msg := "Permission has expired, disconnect"
				klog.Infof("Session[%s] permission has expired, disconnect", s.ID)
				msg = utils.WrapperWarn(msg)
				//replayRecorder.Record([]byte(msg))
				room.Broadcast(&exchange.RoomMessage{Event: exchange.DataEvent, Body: []byte("\n\r" + msg)})
				return
			}
			continue
			// 手动结束
		case <-s.ctx.Done():
			//adminUser := s.loadOperator()
			msg := "Terminated by admin"
			msg = utils.WrapperWarn(msg)
			//replayRecorder.Record([]byte(msg))
			klog.Infof("Session[%s]: %s", s.ID, msg)
			room.Broadcast(&exchange.RoomMessage{Event: exchange.DataEvent, Body: []byte("\n\r" + msg)})
			return
			// 监控窗口大小变化
		case win, ok := <-winCh:
			if !ok {
				return
			}
			_ = srvConn.SetWinSize(win.Width, win.Height)
			klog.Infof("Session[%s] Window server change: %d*%d",
				s.ID, win.Width, win.Height)
			p, _ := json.Marshal(win)
			msg := exchange.RoomMessage{
				Event: exchange.WindowsEvent,
				Body:  p,
			}
			room.Broadcast(&msg)
			// 经过parse处理的server数据，发给user
		case p, ok := <-srvOutChan:
			if !ok {
				return
			}
			//if parser.NeedRecord() {
			//	replayRecorder.Record(p)
			//}
			msg := exchange.RoomMessage{
				Event: exchange.DataEvent,
				Body:  p,
			}
			room.Broadcast(&msg)
			// 经过parse处理的user数据，发给server
		case p, ok := <-userOutChan:
			if !ok {
				return
			}
			if _, err1 := srvConn.Write(p); err1 != nil {
				klog.Errorf("Session[%s] srvConn write err: %s", s.ID, err1)
			}

		case now := <-keepAliveTick.C:
			if now.After(lastActiveTime.Add(keepAliveTime)) {
				if err := srvConn.KeepAlive(); err != nil {
					klog.Errorf("Session[%s] srvCon keep alive err: %s", s.ID, err)
				}
			}
			continue
		case <-userConn.Context().Done():
			klog.Infof("Session[%s]: user conn context done", s.ID)
			return nil
		case <-exitSignal:
			klog.Infof("Session[%s] end by exit signal", s.ID)
			return
		case notifyMsg := <-s.notifyMsgChan:
			klog.Infof("Session[%s] notify event: %s", s.ID, notifyMsg.Event)
			room.Broadcast(notifyMsg)
			continue
		}
		lastActiveTime = time.Now()
	}
}

func ParseStream(userInChan chan *exchange.RoomMessage, srvInChan <-chan []byte, closed <-chan struct{}) (userOut, srvOut <-chan []byte) {
	userOutputChan := make(chan []byte, 1)
	srvOutputChan := make(chan []byte, 1)
	//lastActiveTime := time.Now()
	go func() {
		defer func() {
			close(userOutputChan)
			close(srvOutputChan)
		}()
		for {
			select {
			case <-closed:
				return
			case msg, ok := <-userInChan:
				if !ok {
					return
				}
				var b []byte
				switch msg.Event {
				case exchange.DataEvent:
					b = msg.Body
				}
				if len(b) == 0 {
					continue
				}
				select {
				case <-closed:
					return
				case userOutputChan <- b:
				}

			case b, ok := <-srvInChan:
				if !ok {
					return
				}
				select {
				case <-closed:
					return
				case srvOutputChan <- b:
				}
			}
		}
	}()
	return userOutputChan, srvOutputChan
}
