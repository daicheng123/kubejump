package auth

import (
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/internal/service"
	"github.com/gliderlabs/ssh"
	"k8s.io/klog/v2"
	"net"
	"strings"
)

const (
	ContextKeyUser              = "CONTEXT_USER"
	ContextKeyDirectLoginFormat = "CONTEXT_DIRECT_LOGIN_FORMAT"

	SeparatorATSign   = "@"
	SeparatorHashMark = "#"
)

type LoginAssetReq struct {
	Username    string
	SysUserInfo string
	AssetInfo   string
	Info        *entity.ConnectTokenInfo
}

func (lr *LoginAssetReq) IsToken() bool {
	return lr.Info != nil
}

func (lr *LoginAssetReq) Authenticate(password string) bool {
	return lr.Info.Secret == password
}

type SSHAuthFunc func(ctx ssh.Context, password, publicKey string) (res bool)

func SSHPasswordAndPublicKeyAuth(userService *service.UserService) SSHAuthFunc {
	return func(ctx ssh.Context, password, publicKey string) (res bool) {

		//user, err := userService.GetUserInfoByName(ctx, ctx.User())
		//if err != nil {
		//	klog.Error("fetch userinfo failed, err:[%s]", err.Error())
		//	return false, err
		//}

		remoteAddr, _, _ := net.SplitHostPort(ctx.RemoteAddr().String())
		if req, ok := parseLoginReq(ctx); ok {
			if req.IsToken() && req.Authenticate(password) {
				ctx.SetValue(ContextKeyUser, req.Info.User)
				klog.Infof("SSH conn[%s] %s for %s from %s", ctx.SessionID(),
					actionAccepted, ctx.User(), remoteAddr)
				return true
			}
		}
		return false
	}
}

func parseLoginReq(ctx ssh.Context) (*LoginAssetReq, bool) {
	if req, ok := ctx.Value(ContextKeyDirectLoginFormat).(*LoginAssetReq); ok {
		return req, true
	}

	if req, ok := parseUsernameFormatReq(ctx); ok {
		ctx.SetValue(ContextKeyDirectLoginFormat, req)
		return req, true
	}
	return nil, false
}

func parseUsernameFormatReq(ctx ssh.Context) (*LoginAssetReq, bool) {
	if req, ok := ParseUserFormat(ctx.User()); ok {
		return &req, true
	}
	return nil, false
}

func ParseUserFormat(s string) (LoginAssetReq, bool) {
	for _, separator := range []string{SeparatorATSign, SeparatorHashMark} {
		if req, ok := parseUserFormatBySeparator(s, separator); ok {
			return req, true
		}
	}
	return LoginAssetReq{}, false
}

func parseUserFormatBySeparator(s, Separator string) (LoginAssetReq, bool) {
	authInfos := strings.Split(s, Separator)
	if len(authInfos) != 3 {
		return LoginAssetReq{}, false
	}
	req := LoginAssetReq{
		Username:    authInfos[0],
		SysUserInfo: authInfos[1],
		AssetInfo:   authInfos[2],
	}
	return req, true
}
