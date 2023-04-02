package app

import (
	"github.com/daicheng123/kubejump/config"
	"github.com/daicheng123/kubejump/internal/base/data"
	"github.com/daicheng123/kubejump/internal/entity/utils"
	"github.com/daicheng123/kubejump/internal/repo"
	"github.com/daicheng123/kubejump/internal/service"
	"github.com/daicheng123/kubejump/internal/sshd"
	"k8s.io/klog/v2"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

type server struct {
	terminalConf atomic.Value
	jmsService   *service.JMService
	k8sService   *service.KubernetesService
	userService  *service.UserService
}

func (s *server) run() {
	//wait.WaitFor(func(done <-chan struct{}) <-chan struct{} {
	//
	//}, )
	//for {
	//	if k8sClusterCfgs, err := s.jmsService.ListClusterConfig(); err == nil {
	//		for _, cfg := range k8sClusterCfgs {
	//			s.k8sService.AddSyncResourceToStore(kubernetes.POD_INFORMER_NAME, cfg)
	//
	//		}
	//		break
	//	} else {
	//		time.Sleep(time.Minute)
	//	}
	//
	//}
}

func NewServer(
	jmsService *service.JMService,
	k8sService *service.KubernetesService,
	userService *service.UserService,
) *server {
	app := server{
		jmsService:  jmsService,
		k8sService:  k8sService,
		userService: userService,
	}
	go app.run()
	return &app
}

type JUMP struct {
	sshdSrv *sshd.Server
}

func (j *JUMP) Start() {

	//go k.webSrv.Start()
	go j.sshdSrv.Start()
}

func (j *JUMP) Stop() {
	j.sshdSrv.Stop()
	klog.Info("Quit The KubeJump")
}

func RunForever(confPath string) {
	config.Setup(confPath)
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	appData := data.InitData()
	defer appData.Clean()

	err := utils.InitDBSchema(appData.DB)
	if err != nil {
		klog.Fatalf("init db schema failed, err:[%s]", err.Error())
	}

	// repo层
	userRepo := repo.NewUserRepo()
	clusterRepo := repo.NewClusterRepo()

	k8sService, err := service.NewKubernetesService()
	jmsService := service.NewJMService(clusterRepo, userRepo)
	userService := service.NewUserService(userRepo)

	if err != nil {
		klog.Fatalf("init k8s client factory failed, err:[%s]", err.Error())
	}

	srv := NewServer(jmsService, k8sService, userService)
	sshdSrv := sshd.NewSshServer(srv)
	app := &JUMP{
		sshdSrv: sshdSrv,
	}
	app.Start()
	<-gracefulStop
	app.Stop()
}
