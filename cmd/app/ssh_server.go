package app

import (
	"github.com/daicheng123/kubejump/config"
	"github.com/daicheng123/kubejump/internal/auth"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/internal/sshd"
	"github.com/daicheng123/kubejump/pkg/handler"
	k8sterminal "github.com/daicheng123/kubejump/pkg/kubernetes/terminal"
	"github.com/daicheng123/kubejump/pkg/utils"
	"github.com/gliderlabs/ssh"
	"k8s.io/klog/v2"
	"net"
)

func (s *server) LocalPortForwardingPermission(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
	return config.GlobalConfig.EnableLocalPortForward
}

const ctxID = "ctxID"

func (s *server) PasswordAuth(ctx ssh.Context, password string) bool {
	ctx.SetValue(ctxID, ctx.SessionID())
	return auth.SSHPasswordAndPublicKeyAuth(s.userService)(ctx, password, "")
}

func (s *server) GetSSHSigner() ssh.Signer {
	klog.Infoln(k8sterminal.TERMINAL_HOST_KEY)
	singer, err := sshd.ParsePrivateKeyFromString(k8sterminal.TERMINAL_HOST_KEY)
	if err != nil {
		klog.Fatal(err)
	}
	return singer
}

func (s *server) SessionHandler(sess ssh.Session) {
	user, ok := sess.Context().Value(auth.ContextKeyUser).(*entity.User)
	if !ok || user.ID == 0 {
		klog.Errorf("SSH User %s not found, exit.", sess.User())
		utils.IgnoreErrWriteString(sess, "Not auth user.\n")
		return
	}

	if pty, winChan, isPty := sess.Pty(); isPty {
		interactiveSrv := handler.NewInteractiveHandler(sess, user, s.jmsService)
		klog.Infof("User %s request pty %s", sess.User(), pty.Term)
		go interactiveSrv.WatchWinSizeChange(winChan)
		interactiveSrv.Dispatch()
	}
}

func (s *server) GetSSHAddr() string {
	cf := config.GlobalConfig
	return net.JoinHostPort(cf.BindHost, cf.SSHPort)
}

func (s *server) GetTerminalConfig() entity.TerminalConfig {
	return s.terminalConf.Load().(entity.TerminalConfig)
}

//func buildDirectRequestOptions(userInfo *entity.User, directRequest *auth.LoginAssetReq) []handler.DirectOpt {
//	opts := make([]handler.DirectOpt, 0, 7)
//	opts = append(opts, handler.DirectTargetAsset(directRequest.AssetInfo))
//	opts = append(opts, handler.DirectUser(userInfo))
//	opts = append(opts, handler.DirectTargetSystemUser(directRequest.SysUserInfo))
//	//if directRequest.IsUUIDString() {
//	//	opts = append(opts, handler.DirectFormatType(handler.FormatUUID))
//	//}
//	if directRequest.IsToken() {
//		opts = append(opts, handler.DirectFormatType(handler.FormatToken))
//		opts = append(opts, handler.DirectConnectToken(directRequest.Info))
//	}
//	return opts
//}
//
//func promptAndSelectNames(sess ssh.Session, term *terminal.Terminal, getNames func() ([]string, error)) (string, bool, error) {
//	names, err := getNames()
//	if err != nil {
//		return "", false, err
//	}
//
//	// prompt
//	for i, v := range names {
//		_, _ = fmt.Fprintf(sess, "[%d] %s \r\n", i+1, v)
//	}
//	_, _ = fmt.Fprintln(sess, "Please select ID: ")
//	sel, err := term.ReadLine()
//	if err != nil {
//		return "", false, err
//	}
//
//	if sel == "quit" || sel == "exit" {
//		return "", true, nil
//	}
//
//	i, err := strconv.Atoi(sel)
//	if err != nil {
//		_, _ = fmt.Fprintf(sess, "Invalid input [%s], please again \r\n", sel)
//		return promptAndSelectNames(sess, term, getNames)
//	}
//
//	if (i < 1) || (i > len(names)) {
//		_, _ = fmt.Fprintf(sess, "ID[%d] out of range, please again\r\n", i)
//		return promptAndSelectNames(sess, term, getNames)
//	}
//	klog.Infof("the select pods is %s", names[i-1])
//	return names[i-1], false, nil
//}
