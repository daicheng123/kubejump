package proxy

import (
	"context"
	"github.com/daicheng123/kubejump/pkg/exchange"
	"github.com/gliderlabs/ssh"
	"io"
)

type UserConnection interface {
	io.ReadWriteCloser
	ID() string
	WinCh() <-chan ssh.Window
	LoginFrom() string
	RemoteAddr() string
	Pty() ssh.Pty
	Context() context.Context
	HandleRoomEvent(event string, msg *exchange.RoomMessage)
}
