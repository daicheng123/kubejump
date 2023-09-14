package conn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
	jump_kubernetes "github.com/daicheng123/kubejump/pkg/kubernetes"
	"github.com/toolkits/pkg/logger"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
	"net/http"
	"strings"
	"sync"
)

var (
	ErrNotFoundCommand = errors.New("not found command")

	ErrNotFoundShell = errors.New("not found any shell")
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
type ContainerOption func(*ContainerOptions)

type ContainerOptions struct {
	//UniqKey       string
	Token         string
	PodName       string
	Namespace     string
	ContainerName string
	Cluster       *entity.ClusterConfig
	IsSkipTls     bool
	win           *remotecommand.TerminalSize
}

func (o ContainerOptions) String() string {
	return fmt.Sprintf("(%s)-(%s)-(%s)", o.Namespace,
		o.PodName, o.ContainerName)
}

type ContainerFunc func(*ContainerOptions)

func (cf ContainerFunc) apply(option *ContainerOptions) {
	cf(option)
}

func ContainerClusterConfig(cluster *entity.ClusterConfig) ContainerFunc {
	return func(options *ContainerOptions) {
		options.Cluster = cluster
	}
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

func ContainerSkipTls(isSkipTls bool) ContainerFunc {
	return func(options *ContainerOptions) {
		options.IsSkipTls = isSkipTls
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

	cli, err := factory.GetOrCreateClient(opt.Cluster)
	if err != nil {
		return nil, err
	}

	sehll, err := FindAvailableShell(opt)
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
		shell:        sehll,
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

func (c *ContainerConnection) Read(p []byte) (int, error) {
	return c.stdoutReader.Read(p)
}

func (c *ContainerConnection) Write(p []byte) (int, error) {
	return c.stdinWriter.Write(p)
}

func (c *ContainerConnection) Close() error {
	c.once.Do(func() {
		_, _ = c.stdinWriter.Write([]byte("\r\nexit\r\n"))
		_ = c.stdinWriter.Close()
		_ = c.stdoutReader.Close()
		close(c.done)
		logger.Infof("K8s %s connection close", c.opt.String())
	})
	return nil
}

func (c *ContainerConnection) KeepAlive() error {
	return nil
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

func (c *ContainerConnection) SetWinSize(w, h int) error {
	size := &remotecommand.TerminalSize{
		Width:  uint16(w),
		Height: uint16(h),
	}
	select {
	case c.winSizeChan <- size:
	case <-c.done:
		return nil
	}
	return nil
}

func FindAvailableShell(opt *ContainerOptions) (shell string, err error) {
	shells := []string{"bash", "sh", "powershell", "cmd"}
	for i := range shells {
		if err = HasShellInContainer(opt, shells[i]); err == nil {
			return shells[i], nil
		} else {
			logger.Debug(err)
		}
	}
	return "", ErrNotFoundShell
}

var scriptTmpl = `#!/bin/sh
command -v %s`

func HasShellInContainer(opt *ContainerOptions, shell string) error {
	container := opt.ContainerName
	podName := opt.PodName
	namespace := opt.Namespace
	//kubeConf := opt.K8sCfg()

	factory, err := jump_kubernetes.GetClientFactory()
	if err != nil {
		return err
	}

	client, err := factory.GetOrCreateClient(opt.Cluster)
	if err != nil {
		return err
	}

	testScript := fmt.Sprintf(scriptTmpl, shell)
	command := []string{"sh", "-c", testScript}
	validateChecker := func(result string) error {
		if !strings.HasSuffix(result, shell) {
			return fmt.Errorf("%w: %s %s", ErrNotFoundCommand, result, shell)
		}
		return nil
	}
	switch shell {
	case "cmd":
		command = []string{"where", "cmd"}
		validateChecker = func(result string) error {
			if !strings.HasSuffix(result, "cmd.exe") {
				return fmt.Errorf("%w: %s cmd.exe", ErrNotFoundCommand, result)
			}
			return nil
		}
	case "powershell":
		command = []string{"Get-Command", "powershell"}
		validateChecker = func(result string) error {
			if !strings.Contains(result, "powershell.exe") {
				return fmt.Errorf("%w: %s powershell.exe", ErrNotFoundCommand, result)
			}
			return nil
		}
	}
	req := client.K8sClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdout:    true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(client.GetConfig(), http.MethodPost, req.URL())
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdout: &buf,
		Tty:    false,
	})
	if err != nil {
		return err
	}
	result := strings.TrimSpace(buf.String())
	buf.Reset()
	return validateChecker(result)
}
