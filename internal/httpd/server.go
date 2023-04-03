package httpd

import (
	"context"
	"fmt"
	"github.com/daicheng123/kubejump/internal/service"
	"net/http"
	"time"
)

type Server struct {
	//broadCaster *broadcaster
	Srv        *http.Server
	jmsService *service.JMService
	//JmsService  *service.JMService
}

func NewServer(jmsService *service.JMService) *Server {
	return &Server{
		jmsService: jmsService,
	}
}

func (s *Server) Start() {
	fmt.Println(s.Srv.ListenAndServe())
	//klog.Fatal(s.Srv.ListenAndServe())
}

func (s *Server) Stop() {
	ctx, cancelFunc := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelFunc()
	if s.Srv != nil {
		_ = s.Srv.Shutdown(ctx)
	}
}
