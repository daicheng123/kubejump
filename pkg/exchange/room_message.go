package exchange

import (
	"container/ring"
	"encoding/json"
	"github.com/daicheng123/kubejump/pkg/utils"
	"github.com/toolkits/pkg/logger"
	"io"
	"sort"
	"sync"
	"time"
)

type RoomMessage struct {
	Event string      `json:"event"`
	Body  []byte      `json:"data"`
	Meta  MetaMessage `json:"meta"` // receive的信息必须携带Meta
}

func (m *RoomMessage) Marshal() []byte {
	p, _ := json.Marshal(m)
	return p
}

type MetaMessage struct {
	UserId     string `json:"user_id"`
	User       string `json:"user"`
	Created    string `json:"created"`
	RemoteAddr string `json:"remote_addr"`
	TerminalId string `json:"terminal_id"`
	Primary    bool   `json:"primary"`
	Writable   bool   `json:"writable"`
}

const (
	PingEvent    = "Ping"
	DataEvent    = "Data"
	WindowsEvent = "Windows"

	PauseEvent  = "Pause"
	ResumeEvent = "Resume"

	JoinEvent  = "Join"
	LeaveEvent = "Leave"

	ExitEvent = "Exit"

	JoinSuccessEvent = "JoinSuccess"

	ShareJoin  = "Share_JOIN"
	ShareLeave = "Share_LEAVE"
	ShareUsers = "Share_USERS"

	ActionEvent = "Action"

	ShareRemoveUser = "Share_REMOVE_USER"
)

const (
	ZmodemStartEvent = "ZMODEM_START"
	ZmodemEndEvent   = "ZMODEM_END"
)

type RoomManager interface {
	Add(s *Room)
	Delete(s *Room)
	Get(sid string) *Room
}

var (
	_ RoomManager = (*localRoomManager)(nil)
	_ RoomManager = (*redisRoomManager)(nil)
)

func CreateRoom(id string, inChan chan *RoomMessage) *Room {
	s := &Room{
		Id:             id,
		userInputChan:  inChan,
		broadcastChan:  make(chan *RoomMessage),
		subscriber:     make(chan *Conn),
		unSubscriber:   make(chan *Conn),
		exitSignal:     make(chan struct{}),
		done:           make(chan struct{}),
		recentMessages: ring.New(5),
	}
	return s
}

type Room struct {
	Id string

	userInputChan chan *RoomMessage

	broadcastChan chan *RoomMessage

	subscriber chan *Conn

	unSubscriber chan *Conn

	exitSignal chan struct{}

	done chan struct{}

	once sync.Once

	recentMessages *ring.Ring
}

func (r *Room) run() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	defer r.closeOnce()
	connMaps := make(map[string]*Conn)
	currentOnlineUsers := make(map[string]MetaMessage)
	var ZMODEMStatus bool
	for {
		select {
		case <-ticker.C:
			if len(connMaps) == 0 {
				logger.Infof("Room %s has no connection now and exit", r.Id)
				return
			}
			select {
			case <-r.done:
				for k := range connMaps {
					_ = connMaps[k].Close()
				}
			default:
			}
		case con := <-r.subscriber:
			connMaps[con.Id] = con
			if ZMODEMStatus {
				con.handlerMessage(&RoomMessage{
					Event: ActionEvent,
					Body:  []byte(ZmodemStartEvent),
				})
			}
			r.recentMessages.Do(func(value interface{}) {
				if msg, ok := value.(*RoomMessage); ok {
					switch msg.Event {
					case DataEvent:
						_, _ = con.Write(msg.Body)
					}
				}
			})
			body, _ := json.Marshal(currentOnlineUsers)
			con.handlerMessage(&RoomMessage{
				Event: ShareUsers,
				Body:  body,
			})
			logger.Debugf("Room %s current connections count: %d", r.Id, len(connMaps))
		case con := <-r.unSubscriber:
			delete(connMaps, con.Id)
			logger.Debugf("Room %s current connections count: %d", r.Id, len(connMaps))
		case msg := <-r.broadcastChan:
			userCones := make([]*Conn, 0, len(connMaps))
			for k := range connMaps {
				userCones = append(userCones, connMaps[k])
			}
			switch msg.Event {
			case DataEvent:
				r.recentMessages.Value = msg
				r.recentMessages = r.recentMessages.Next()
			case ShareJoin:
				key := msg.Meta.User + msg.Meta.Created
				currentOnlineUsers[key] = msg.Meta
			case ShareLeave:
				key := msg.Meta.User + msg.Meta.Created
				delete(currentOnlineUsers, key)
			case ActionEvent:
				switch string(msg.Body) {
				case ZmodemStartEvent:
					ZMODEMStatus = true
				case ZmodemEndEvent:
					ZMODEMStatus = false
				default:
					ZMODEMStatus = false
				}
			}
			r.broadcastMessage(userCones, msg)

		case <-r.exitSignal:
			for k := range connMaps {
				_ = connMaps[k].Close()
			}
		}
	}
}

func (r *Room) Subscribe(conn *Conn) {
	r.subscriber <- conn

}

func (r *Room) UnSubscribe(conn *Conn) {
	r.unSubscriber <- conn
}

func (r *Room) Broadcast(msg *RoomMessage) {
	select {
	case <-r.done:
	case r.broadcastChan <- msg:
	}
}

func (r *Room) Receive(msg *RoomMessage) {
	select {
	case <-r.done:
	case r.userInputChan <- msg:
	}
}

func (r *Room) broadcastMessage(conns userConnections, msg *RoomMessage) {
	// 减少启动goroutine的数量
	if len(conns) == 0 {
		return
	}
	if len(conns) == 1 {
		conns[0].handlerMessage(msg)
		return
	}

	// 启动 goroutine 发送消息
	sort.Sort(conns)
	var wg sync.WaitGroup
	for i := range conns {
		wg.Add(1)
		go func(con *Conn) {
			defer wg.Done()
			con.handlerMessage(msg)
		}(conns[i])
	}
	wg.Wait()
}

func (r *Room) Done() <-chan struct{} {
	return r.done
}

func (r *Room) stop() {
	select {
	case <-r.done:
		return
	case r.exitSignal <- struct{}{}:
	}
	r.closeOnce()
}

func (r *Room) closeOnce() {
	r.once.Do(func() {
		close(r.done)
	})
}

func WrapperUserCon(stream Stream) *Conn {
	return &Conn{
		Id:      utils.UUID(),
		Stream:  stream,
		created: time.Now(),
	}
}

type Stream interface {
	io.WriteCloser
	HandleRoomEvent(event string, msg *RoomMessage)
}

type Conn struct {
	Id string
	Stream
	created time.Time
}

func (c *Conn) handlerMessage(msg *RoomMessage) {
	switch msg.Event {
	case DataEvent:
		_, _ = c.Write(msg.Body)
	case PingEvent:
		_, _ = c.Write(nil)
	default:
		c.HandleRoomEvent(msg.Event, msg)
	}
}

var _ sort.Interface = (userConnections)(nil)

type userConnections []*Conn

func (l userConnections) Less(i, j int) bool {
	return l[i].created.Before(l[j].created)
}

func (l userConnections) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l userConnections) Len() int {
	return len(l)
}
