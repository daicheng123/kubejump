package app

import (
	"fmt"
	"github.com/daicheng123/kubejump/config"
	"github.com/daicheng123/kubejump/internal/auth"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/internal/sshd"
	k8sterminal "github.com/daicheng123/kubejump/pkg/kubernetes/terminal"
	"github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/klog/v2"
	"net"
	"strconv"
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
	//user, ok := sess.Context().Value(auth.ContextKeyUser).(*entity.User)
	//if !ok || user.ID == 0 {
	//	klog.Errorf("SSH User %s not found, exit.", sess.User())
	//	utils.IgnoreErrWriteString(sess, "Not auth user.\n")
	//	return
	//}

	defer func() {
		sess.Exit(0)
	}()
	term := terminal.NewTerminal(sess, "")

	c, err := s.jmsService.GetKubernetesCfg(1)
	if err != nil {
		klog.Errorf("[ERROR]: %s", err.Error())
		_, _ = fmt.Fprintf(sess, "[ERROR]: %s\n", err.Error())
		return
	}

	ns, exit, err := promptAndSelectNames(sess, term, func() ([]string, error) {
		nl, err := s.k8sService.ListNamespaces(sess.Context(), c)
		if err != nil {
			return nil, err
		}
		var names = make([]string, 0, len(nl.Items))
		for _, item := range nl.Items {
			names = append(names, item.Name)
		}
		return names, err
	})

	if err != nil {
		klog.Error(err.Error())
		_, _ = fmt.Fprintf(sess, "[ERROR]: %v", err)
		return
	}
	if exit {
		return
	}

	selectPod, exit, err := promptAndSelectNames(sess, term, func() ([]string, error) {
		pl, err := s.k8sService.ListPodsByNamespace(sess.Context(), ns, c)
		if err != nil {
			return nil, err
		}
		var pods = make([]string, 0, len(pl.Items))
		for _, item := range pl.Items {
			pods = append(pods, item.Name)
		}
		return pods, nil
	})

	if err != nil {
		_, _ = fmt.Fprintf(sess, "[ERROR]: %s", err.Error())
		return
	}

	if exit {
		return
	}

	kts := k8sterminal.TerminalSession{SSHSession: sess}

	if err = k8sterminal.StartProcess(sess.Context(), c, []string{"/bin/sh"}, kts, ns, selectPod, ""); err != nil {
		klog.Error("==>>k8s.StartProcess", err)
		_, _ = fmt.Fprintf(sess, "<<==>>k8s.StartProcess:%v\r\n", err)
		//sess.Exit(0)
		return
	}
}

func (s *server) GetSSHAddr() string {
	cf := config.GlobalConfig
	return net.JoinHostPort(cf.BindHost, cf.SSHPort)
}

func (s *server) GetTerminalConfig() entity.TerminalConfig {
	return s.terminalConf.Load().(entity.TerminalConfig)
}

func promptAndSelectNames(sess ssh.Session, term *terminal.Terminal, getNames func() ([]string, error)) (string, bool, error) {
	names, err := getNames()
	if err != nil {
		return "", false, err
	}

	// prompt
	for i, v := range names {
		_, _ = fmt.Fprintf(sess, "[%d] %s \r\n", i+1, v)
	}
	_, _ = fmt.Fprintln(sess, "Please select ID: ")
	sel, err := term.ReadLine()
	if err != nil {
		return "", false, err
	}

	if sel == "quit" || sel == "exit" {
		return "", true, nil
	}

	i, err := strconv.Atoi(sel)
	if err != nil {
		_, _ = fmt.Fprintf(sess, "Invalid input [%s], please again \r\n", sel)
		return promptAndSelectNames(sess, term, getNames)
	}

	if (i < 1) || (i > len(names)) {
		_, _ = fmt.Fprintf(sess, "ID[%d] out of range, please again\r\n", i)
		return promptAndSelectNames(sess, term, getNames)
	}
	klog.Infof("the select pods is %s", names[i-1])
	return names[i-1], false, nil
}
