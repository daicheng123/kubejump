package terminal

import (
	"github.com/gliderlabs/ssh"
	"io"
	"k8s.io/client-go/tools/remotecommand"
)

type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

type TerminalSession struct {
	SSHSession ssh.Session
}

func (t TerminalSession) Next() *remotecommand.TerminalSize {
	_, winChan, _ := t.SSHSession.Pty()
	select {
	case win := <-winChan:
		return &remotecommand.TerminalSize{Width: uint16(win.Width), Height: uint16(win.Height)}
	}
}

func (t TerminalSession) Read(p []byte) (int, error) {
	return t.SSHSession.Read(p)
}

func (t TerminalSession) Write(p []byte) (int, error) {
	return t.SSHSession.Write(p)
}
