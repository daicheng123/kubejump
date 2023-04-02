package sshd

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/pires/go-proxyproto"
	"k8s.io/klog/v2"
	"net"
	"time"
)

type Server struct {
	Srv *ssh.Server
}

func (s *Server) Start() {
	klog.Infof("Start sshd server at %s", s.Srv.Addr)
	ln, err := net.Listen("tcp", s.Srv.Addr)
	if err != nil {
		klog.Fatal(err)
	}
	proxyListener := &proxyproto.Listener{Listener: ln}
	klog.Fatal(s.Srv.Serve(proxyListener))
}

func (s *Server) Stop() {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	klog.Fatal(s.Srv.Shutdown(ctx))

}

type SSHHandler interface {
	LocalPortForwardingPermission(ctx ssh.Context, destinationHost string, destinationPort uint32) bool
	GetSSHSigner() ssh.Signer
	PasswordAuth(ctx ssh.Context, password string) bool
	SessionHandler(ssh.Session)
	GetSSHAddr() string
}

func NewSshServer(handler SSHHandler) *Server {
	srv := &ssh.Server{

		Addr: handler.GetSSHAddr(),
		//LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
		//	return handler.LocalPortForwardingPermission(ctx, destinationHost, destinationPort)
		//},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			return password == "secret"
		},

		HostSigners: []ssh.Signer{handler.GetSSHSigner()},

		Handler: handler.SessionHandler,
	}
	return &Server{Srv: srv}
}
