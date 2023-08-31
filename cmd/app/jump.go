package app

import (
	"github.com/daicheng123/kubejump/config"
	"github.com/daicheng123/kubejump/internal/base/data"
	"github.com/daicheng123/kubejump/internal/entity/utils"
	"github.com/daicheng123/kubejump/internal/httpd"
	"github.com/daicheng123/kubejump/internal/repo"
	"github.com/daicheng123/kubejump/internal/service"
	"github.com/daicheng123/kubejump/internal/sshd"
	"github.com/daicheng123/kubejump/pkg/api"
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
	webSrv  *httpd.Server
}

func (j *JUMP) Start() {

	go j.webSrv.Start()
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

	// repo
	userRepo := repo.NewUserRepo()
	clusterRepo := repo.NewClusterRepo()
	podRepo := repo.NewPodRepo()
	nsRepo := repo.NewNamespaceRepo()

	// service
	k8sService, err := service.NewKubernetesService(podRepo, nsRepo)
	jmsService := service.NewJMService(clusterRepo, userRepo, podRepo)
	userService := service.NewUserService(userRepo)

	if err != nil {
		klog.Fatalf("init k8s client factory failed, err:[%s]", err.Error())
	}

	srv := NewServer(jmsService, k8sService, userService)

	// start event task
	go syncClusterResourcesToStore(srv)

	webSrv := httpd.NewServer(jmsService)
	api.RegisterWebHandler(webSrv)
	sshdSrv := sshd.NewSshServer(srv)
	app := &JUMP{
		sshdSrv: sshdSrv,
		webSrv:  webSrv,
	}
	app.Start()
	<-gracefulStop
	app.Stop()
}
