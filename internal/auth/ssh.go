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

	tokenPrefix = "JUMP-"
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
		remoteAddr, _, _ := net.SplitHostPort(ctx.RemoteAddr().String())

		authMethod := "publickey"
		if password != "" {
			authMethod = "password"
		}

		if req, ok := parseLoginReq(userService, ctx); ok {
			if req.IsToken() && req.Authenticate(password) {
				ctx.SetValue(ContextKeyUser, req.Info.User)
				klog.Infof("SSH conn[%s] authenticating user %s %s from %s", ctx.SessionID(), ctx.User(), authMethod, remoteAddr)
				return true
			}
		}
		return false
	}
}

func parseLoginReq(userService *service.UserService, ctx ssh.Context) (*LoginAssetReq, bool) {
	if req, ok := ctx.Value(ContextKeyDirectLoginFormat).(*LoginAssetReq); ok {
		return req, true
	}

	if req, ok := parseJMSTokenLoginReq(userService, ctx); ok {
		//ctx.SetValue(ContextKeyDirectLoginFormat, req)
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

func parseJMSTokenLoginReq(userService *service.UserService, ctx ssh.Context) (*LoginAssetReq, bool) {
	if userInfo, err := userService.GetUserInfoByName(ctx, ctx.User()); err == nil {
		var assetReq = &LoginAssetReq{
			Username:    userInfo.Username,
			SysUserInfo: userInfo.Name,
			Info: &entity.ConnectTokenInfo{
				User:   userInfo,
				Secret: userInfo.Password,
			},
		}
		return assetReq, true
	} else {
		klog.Errorf("Check user token %s failed: %s", ctx.User(), err)
	}
	return nil, false
}

func parsePasswordOrPublicKeyLoginReq(userService *service.UserService, ctx ssh.Context) (*LoginAssetReq, bool) {
	if userInfo, err := userService.GetUserInfoByName(ctx, ctx.User()); err == nil {
		var assetReq = &LoginAssetReq{
			Username:    userInfo.Username,
			SysUserInfo: userInfo.Name,
			Info: &entity.ConnectTokenInfo{
				User:   userInfo,
				Secret: userInfo.Password,
			},
		}
		return assetReq, true
	} else {
		klog.Errorf("Check user token %s failed: %s", ctx.User(), err)
	}
	return nil, false
}
