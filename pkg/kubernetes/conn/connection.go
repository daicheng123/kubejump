package conn

import (
	"context"
	"github.com/daicheng123/kubejump/internal/entity"
	jump_kubernetes "github.com/daicheng123/kubejump/pkg/kubernetes"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
	"net/http"
	"sync"
)

type ServerConnection interface {
	io.ReadWriteCloser
	SetWinSize(width, height int) error
	KeepAlive() error
}

type Windows struct {
	Width  int
	Height int
}

type ContainerOptions struct {
	Host          string
	Token         string
	PodName       string
	Namespace     string
	ContainerName string
	IsSkipTls     bool
	win           *remotecommand.TerminalSize
}

type ContainerFunc func(*ContainerOptions)

func (cf ContainerFunc) apply(option *ContainerOptions) {
	cf(option)
}

func ContainerPodName(podName string) ContainerFunc {
	return func(options *ContainerOptions) {
		options.PodName = podName
	}
}

func ContainerNamespace(namespace string) ContainerFunc {
	return func(opt *ContainerOptions) {
		opt.Namespace = namespace
	}
}

func ContainerName(container string) ContainerFunc {
	return func(opt *ContainerOptions) {
		opt.ContainerName = container
	}
}

func ContainerPtyWin(win Windows) ContainerFunc {
	return func(args *ContainerOptions) {
		args.win = &remotecommand.TerminalSize{
			Width:  uint16(win.Width),
			Height: uint16(win.Height),
		}
	}
}

type ContainerConnection struct {
	opt   *ContainerOptions
	shell string

	slaver      *slaveStream
	winSizeChan chan *remotecommand.TerminalSize

	stdinWriter  io.WriteCloser
	stdoutReader io.ReadCloser

	done chan struct{}

	once sync.Once
}

func (c *ContainerConnection) Close() error {
	c.once.Do(func() {
		_, _ = c.stdinWriter.Write([]byte("\r\nexit\r\n"))
		_ = c.stdinWriter.Close()
		_ = c.stdoutReader.Close()
		close(c.done)
		klog.Infof("K8s %s connection close", c.opt.String())
	})
	return nil
}

func NewKubernetesConnection(options ...ContainerFunc) (*ContainerConnection, error) {
	var opt *ContainerOptions
	for _, setter := range options {
		setter.apply(opt)
	}

	if opt.win == nil {
		opt.win = &remotecommand.TerminalSize{
			Width:  80,
			Height: 40,
		}
	}
	factory, err := jump_kubernetes.GetClientFactory()
	if err != nil {
		return nil, err
	}

	cli, err := factory.GetOrCreateClient(&entity.ClusterConfig{})
	if err != nil {
		return nil, err
	}

	winSizeChan := make(chan *remotecommand.TerminalSize, 10)

	done := make(chan struct{})
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	slaver := slaveStream{
		r:           stdinReader,
		w:           stdoutWriter,
		winSizeChan: winSizeChan,
		done:        done,
	}

	con := ContainerConnection{
		opt:          opt,
		shell:        "sh",
		slaver:       &slaver,
		winSizeChan:  winSizeChan,
		done:         done,
		stdoutReader: stdoutReader,
		stdinWriter:  stdinWriter,
	}

	con.winSizeChan <- opt.win
	go func() {
		if err2 := execContainerShell(cli, &con); err2 != nil {
			klog.Error(err2)
		}
		_ = con.Close()
		klog.Infof("K8s %s exec shell exit", con.opt.String())
	}()
	return &con, nil
}

type slaveStream struct {
	r           io.ReadCloser
	w           io.WriteCloser
	winSizeChan chan *remotecommand.TerminalSize
	done        chan struct{}
}

func (s *slaveStream) Read(p []byte) (int, error) {
	return s.r.Read(p)
}

func (s *slaveStream) Write(p []byte) (int, error) {
	return s.w.Write(p)
}

func (s *slaveStream) Next() *remotecommand.TerminalSize {
	select {
	case size := <-s.winSizeChan:
		return size
	case <-s.done:
		return nil
	}
}

func execContainerShell(k8sClient *jump_kubernetes.ClientSet, c *ContainerConnection) error {
	req := k8sClient.K8sClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(c.opt.PodName).
		Namespace(c.opt.Namespace).
		SubResource("exec")
	req.VersionedParams(&v1.PodExecOptions{
		Container: c.opt.ContainerName,
		Command:   []string{c.shell},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(k8sClient.GetConfig(), http.MethodPost, req.URL())
	if err != nil {
		return err
	}
	streamOption := remotecommand.StreamOptions{
		Stdin:             c.slaver,
		Stdout:            c.slaver,
		Stderr:            c.slaver,
		TerminalSizeQueue: c.slaver,
		Tty:               true,
	}
	// 这个 stream 是阻塞的方法
	err = exec.StreamWithContext(context.Background(), streamOption)
	return err
}

var scriptTmpl = `#!/bin/sh
command -v %s`
