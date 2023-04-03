package api

import (
	"github.com/daicheng123/kubejump/config"
	"github.com/daicheng123/kubejump/internal/httpd"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"net"
	"net/http"
)

func RegisterWebHandler(handler *httpd.Server) {
	if config.GetConf().LogLevel != "DEBUG" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	//engine
	trustedProxies := []string{"0.0.0.0/0", "::/0"}
	if err := engine.SetTrustedProxies(trustedProxies); err != nil {
		klog.Fatal(err.Error())
	}

	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	rootGroup := engine.Group("")

	jumpGroup := rootGroup.Group("/api")

	jumpGroup.Handle(http.MethodGet, "health", handler.HealthCheck)

	jumpGroup.Handle(http.MethodPost, "k8s_cluster", handler.ApplyK8sCluster)

	conf := config.GetConf()
	addr := net.JoinHostPort(conf.BindHost, conf.HTTPPort)

	handler.Srv = &http.Server{
		Addr:    addr,
		Handler: engine,
	}
}
