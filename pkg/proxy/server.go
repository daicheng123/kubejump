package proxy

import (
	"context"
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/internal/service"
	"github.com/daicheng123/kubejump/pkg/exchange"
	"github.com/daicheng123/kubejump/pkg/kubernetes/conn"
	"github.com/daicheng123/kubejump/pkg/session"
	"github.com/daicheng123/kubejump/pkg/srvconn"
	"github.com/daicheng123/kubejump/pkg/utils"
	"github.com/toolkits/pkg/logger"
	gossh "golang.org/x/crypto/ssh"
	"k8s.io/klog/v2"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"
)

func NewProxyServer(conn UserConnection, jmsService *service.JMService, opts ...ConnectionOption) (*ProxyServer, error) {
	connOpts := &ConnectionOptions{}
	for _, setter := range opts {
		setter(connOpts)
	}
	asset := connOpts.authInfo.Asset
	user := connOpts.authInfo.User

	//terminalConf, err := jmsService.GetTerminalConfig()

	//if connOpts.k8sContainer != nil {
	//	connOpts.k8sContainer.K8sName()
	//}
	assetName := asset.String()
	//if connOpts.k8sContainer != nil {
	//	assetName = connOpts.k8sContainer.K8sName(asset.Name)
	//}
	apiSession := &entity.Session{
		ID:         utils.UUID(),
		User:       user.String(),
		LoginFrom:  entity.LabelField(conn.LoginFrom()),
		RemoteAddr: conn.RemoteAddr(),
		UserID:     int(user.ID),
		Asset:      assetName,
		AssetID:    int(asset.ID),
		Type:       entity.NORMALType,
	}

	return &ProxyServer{
		ID:          apiSession.ID,
		UserConn:    conn,
		jmsService:  jmsService,
		connOpts:    connOpts,
		sessionInfo: apiSession,
	}, nil
}

type ProxyServer struct {
	ID                 string
	UserConn           UserConnection
	jmsService         *service.JMService
	connOpts           *ConnectionOptions
	terminalConf       *entity.TerminalConfig
	gateway            *entity.Gateway
	domainGateways     *entity.Domain
	sessionInfo        *entity.Session
	cacheSSHConnection *srvconn.SSHConnection
	keyboardMode       int32
	OnSessionInfo      func(info *SessionInfo)
	BroadcastEvent     func(event *exchange.RoomMessage)
}

func (s *ProxyServer) Proxy() {
	defer func() {
		if s.cacheSSHConnection != nil {
			_ = s.cacheSSHConnection.Close()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())

	maxIdleTime := s.terminalConf.MaxIdleTime

	maxSessionTime := time.Now().Add(time.Duration(s.terminalConf.MaxSessionTime) * time.Hour)

	sw := SwitchSession{
		ID:            s.ID,
		MaxIdleTime:   maxIdleTime,
		keepAliveTime: 60,
		ctx:           ctx,
		cancel:        cancel,
		proxy:         s,
		notifyMsgChan: make(chan *exchange.RoomMessage, 1),

		MaxSessionTime: maxSessionTime,
	}

	traceSession := session.NewSession(sw.proxy.sessionInfo, func(task *entity.TerminalTask) error {
		switch task.Name {
		case entity.TaskKillSession:
			sw.Terminate(task.Kwargs.TerminatedBy)
		case entity.TaskLockSession:
			sw.PauseOperation(task.Kwargs.CreatedByUser)
		case entity.TaskUnlockSession:
			sw.ResumeOperation(task.Kwargs.CreatedByUser)
		default:
			return fmt.Errorf("ssh session unknown task %s", task.Name)
		}
		return nil
	})
	session.AddSession(traceSession)
	defer session.RemoveSession(traceSession)

	//var proxyAddr *net.TCPAddr
	//if (s.domainGateways != nil && len(s.domainGateways.Gateways) != 0) || s.gateway != nil {
	//	dGateway := s.createAvailableGateWay(s.domainGateways)
	//	err := dGateway.Start()
	//	if err != nil {
	//		msg := "Start domain gateway failed %s"
	//		msg = fmt.Sprintf(msg, err)
	//		utils.IgnoreErrWriteString(s.UserConn, utils.WrapperWarn(msg))
	//		klog.Error(msg)
	//		return
	//	}
	//	defer dGateway.Stop()
	//	proxyAddr = dGateway.GetListenAddr()
	//}

	srvCon, err := s.getServerConn()
	if err != nil {
		logger.Error(err)
		//s.sendConnectErrorMsg(err)
		//if err2 := s.ConnectedFailedCallback(err); err2 != nil {
		//	logger.Errorf("Conn[%s] update session err: %s", s.UserConn.ID(), err2)
		//}
		return
	}
	defer srvCon.Close()

	klog.Infof("Conn[%s] create session %s success", s.UserConn.ID(), s.ID)
	//if err2 := s.ConnectedSuccessCallback(); err2 != nil {
	//	logger.Errorf("Conn[%s] update session %s err: %s", s.UserConn.ID(), s.ID, err2)
	//}
	if s.OnSessionInfo != nil {
		//actions := s.connOpts.authInfo.Actions
		tokenConnOpts := s.connOpts.authInfo.ConnectOptions
		ctrlCAsCtrlZ := false
		//isK8s := s.connOpts.authInfo.Protocol == srvconn.ProtocolK8s
		isNotPod := s.connOpts.k8sContainer == nil
		if isNotPod {
			ctrlCAsCtrlZ = true
		}
		//perm := actions.Permission()

		info := SessionInfo{
			Session: s.sessionInfo,
			//Perms:   &perm,
			BackspaceAsCtrlH: tokenConnOpts.BackspaceAsCtrlH,
			CtrlCAsCtrlZ:     ctrlCAsCtrlZ,
		}
		go s.OnSessionInfo(&info)
	}

	//utils.IgnoreErrWriteWindowTitle(s.UserConn, s.connOpts.TerminalTitle())
	if err = sw.Bridge(s.UserConn, srvCon); err != nil {
		logger.Error(err)
	}
}

//func (s *ProxyServer) createAvailableGateWay(domain *entity.Domain) *domainGateway {
//	asset := s.connOpts.authInfo.Asset
//	var dGateway *domainGateway
//
//	dGateway = &domainGateway{
//		domain:          domain,
//		dstIP:           asset.PodIP,
//		dstPort:         22,
//		selectedGateway: s.gateway,
//	}
//	return dGateway
//}

type SessionInfo struct {
	Session *entity.Session `json:"session"`
	//Perms   *entity.Permission `json:"permission"`
	BackspaceAsCtrlH *bool `json:"backspaceAsCtrlH,omitempty"`
	CtrlCAsCtrlZ     bool  `json:"ctrlCAsCtrlZ"`
}

type domainGateway struct {
	domain  *entity.Domain
	dstIP   string
	dstPort int

	sshClient       *gossh.Client
	selectedGateway *entity.Gateway
	ln              net.Listener
	once            sync.Once
}

func (s *ProxyServer) getServerConn() (srvconn.ServerConnection, error) {
	if s.cacheSSHConnection != nil {
		return s.cacheSSHConnection, nil
	}
	done := make(chan struct{})
	defer func() {
		utils.IgnoreErrWriteString(s.UserConn, "\r\n")
		close(done)
	}()

	//go s.sendConnectingMsg(done)

	return s.getK8sConConn()

}

//
//func (s *Server) sendConnectingMsg(done chan struct{}) {
//	delay := 0.0
//	maxDelay := 5 * 60.0 // 最多执行五分钟
//	msg := fmt.Sprintf("%s  %.1f", s.connOpts.ConnectMsg(), delay)
//	utils.IgnoreErrWriteString(s.UserConn, msg)
//	var activeFlag bool
//	for delay < maxDelay {
//		select {
//		case <-done:
//			return
//		default:
//			if s.IsKeyboardMode() {
//				activeFlag = true
//				break
//			}
//			if activeFlag {
//				utils.IgnoreErrWriteString(s.UserConn, utils.CharClear)
//				msg = fmt.Sprintf("%s  %.1f", s.connOpts.ConnectMsg(), delay)
//				utils.IgnoreErrWriteString(s.UserConn, msg)
//				activeFlag = false
//				break
//			}
//			delayS := fmt.Sprintf("%.1f", delay)
//			data := strings.Repeat("\x08", len(delayS)) + delayS
//			utils.IgnoreErrWriteString(s.UserConn, data)
//		}
//		time.Sleep(100 * time.Millisecond)
//		delay += 0.1
//	}
//}

// getSSHConn 获取ssh连接
func (s *ProxyServer) getK8sConConn() (srvConn srvconn.ServerConnection, err error) {
	asset := s.connOpts.authInfo.Asset
	cluster := asset.Cluster
	//if localTunnelAddr != nil {
	//	//originUrl, err := url.Parse(clusterServer)
	//	if err != nil {
	//		return nil, err
	//	}
	//clusterServer = ReplaceURLHostAndPort(originUrl, "127.0.0.1", localTunnelAddr.Port)
	//}
	return s.getContainerConn(cluster)
	//if s.connOpts.k8sContainer != nil {
	//
	//}
	//srvConn, err = conn.NewKubernetesConnection(
	//	srvconn.K8sToken(s.account.Secret),
	//	srvconn.K8sClusterServer(clusterServer),
	//	srvconn.K8sUsername(s.account.Username),
	//	srvconn.K8sSkipTls(true),
	//	srvconn.K8sPtyWin(srvconn.Windows{
	//		Width:  s.UserConn.Pty().Window.Width,
	//		Height: s.UserConn.Pty().Window.Height,
	//	}),
	//	srvconn.K8sExtraEnvs(map[string]string{
	//		"K8sName": asset.Name,
	//	}),
	//)
	//return
}

func ReplaceURLHostAndPort(originUrl *url.URL, ip string, port int) string {
	newHost := net.JoinHostPort(ip, strconv.Itoa(port))
	switch originUrl.Scheme {
	case "https":
		if port == 443 {
			newHost = ip
		}
	default:
		if port == 80 {
			newHost = ip
		}
	}
	newUrl := url.URL{
		Scheme:     originUrl.Scheme,
		Opaque:     originUrl.Opaque,
		User:       originUrl.User,
		Host:       newHost,
		Path:       originUrl.Path,
		RawPath:    originUrl.RawQuery,
		ForceQuery: originUrl.ForceQuery,
		RawQuery:   originUrl.RawQuery,
		Fragment:   originUrl.Fragment,
	}
	return newUrl.String()
}

func (s *ProxyServer) getContainerConn(cluster *entity.ClusterConfig) (srvConn srvconn.ServerConnection, err error) {
	info := s.connOpts.k8sContainer
	//token := s.account.Secret
	pty := s.UserConn.Pty()

	win := conn.Windows{
		Width:  pty.Window.Width,
		Height: pty.Window.Height,
	}
	opts := make([]conn.ContainerFunc, 0, 5)
	opts = append(opts, conn.ContainerClusterConfig(cluster))
	opts = append(opts, conn.ContainerName(info.Container))
	opts = append(opts, conn.ContainerPodName(info.PodName))
	opts = append(opts, conn.ContainerNamespace(info.Namespace))
	opts = append(opts, conn.ContainerSkipTls(true))
	opts = append(opts, conn.ContainerPtyWin(win))
	srvConn, err = conn.NewKubernetesConnection(opts...)
	return
}

func (s *ProxyServer) CheckPermissionExpired(now time.Time) bool {
	return s.connOpts.authInfo.ExpireAt.IsExpired(now)
}

//func (s *ProxyServer) GetFilterParser() *Parser {
//	var (
//		enableUpload   bool
//		enableDownload bool
//	)
//	actions := s.connOpts.authInfo.Actions
//	if actions.EnableDownload() {
//		enableDownload = true
//	}
//	if actions.EnableUpload() {
//		enableUpload = true
//	}
//	zParser := zmodem.New()
//	zParser.FileEventCallback = s.ZmodemFileTransferEvent
//	protocol := s.connOpts.authInfo.Protocol
//	filterRules := s.connOpts.authInfo.CommandFilterACLs
//	platform := s.connOpts.authInfo.Platform
//	// 过滤规则排序
//	sort.Sort(model.CommandACLs(filterRules))
//	parser := Parser{
//		id:             s.ID,
//		protocolType:   protocol,
//		jmsService:     s.jmsService,
//		cmdFilterACLs:  filterRules,
//		enableDownload: enableDownload,
//		enableUpload:   enableUpload,
//		zmodemParser:   zParser,
//		i18nLang:       s.connOpts.i18nLang,
//		platform:       &platform,
//	}
//	parser.initial()
//	return &parser
//}
