package srvconn

import (
	"io"

	gossh "golang.org/x/crypto/ssh"
)

type SSHConnection struct {
	session *gossh.Session
	stdin   io.Writer
	stdout  io.Reader
	options *SSHOptions
}

func (sc *SSHConnection) SetWinSize(w, h int) error {
	return sc.session.WindowChange(h, w)
}

func (sc *SSHConnection) Read(p []byte) (n int, err error) {
	return sc.stdout.Read(p)
}

func (sc *SSHConnection) Write(p []byte) (n int, err error) {
	return sc.stdin.Write(p)
}

func (sc *SSHConnection) Close() (err error) {
	return sc.session.Close()
}

func (sc *SSHConnection) KeepAlive() error {
	_, err := sc.session.SendRequest("keepalive@openssh.com", false, nil)
	return err
}

type SSHOption func(*SSHOptions)

type SSHOptions struct {
	charset string
	win     Windows
	term    string

	//suConfig *SuConfig
}

func SSHCharset(charset string) SSHOption {
	return func(opt *SSHOptions) {
		opt.charset = charset
	}
}

func SSHPtyWin(win Windows) SSHOption {
	return func(opt *SSHOptions) {
		opt.win = win
	}
}

func SSHTerm(termType string) SSHOption {
	return func(opt *SSHOptions) {
		opt.term = termType
	}
}

//
//func SSHSudoConfig(cfg *SuConfig) SSHOption {
//	return func(opt *SSHOptions) {
//		opt.suConfig = cfg
//	}
//}
