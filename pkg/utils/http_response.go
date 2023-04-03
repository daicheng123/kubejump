package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Response struct {
	Code   int         `json:"err_code"`
	ErrMsg string      `json:"err_msg"`
	Data   interface{} `json:"data"`
	Msg    string      `json:"msg"`
}

const (
	OkMsg    = "操作成功"
	NotOkMsg = "操作失败"

	ParamErrorMsg = "参数绑定失败, 请检查数据类型"

	//LoginCheckErrorMsg     = "用户名或密码错误"
	//UserRegisterFailMsg    = "用户注册失败"
	//UserNameEmptyMsg       = "用户不能为空"
	//UserPassEmptyMsg       = "密码不能为空"
	//UserDisableMsg         = "用户已被禁用"
	//ForbiddenMsg           = "无权访问该资源"
	InternalServerErrorMsg = "服务器内部错误"

	CreateK8SClusterErrorMsg = "创建K8S集群失败"

	//LDAPUserLoginFailedMsg = "登录失败，请检查您的用户名和密码!"
	//LDAPUserNotFoundMsg    = "用户不存在"
)

var CustomError = map[int]string{
	SUCCESS: OkMsg,
	ERROR:   NotOkMsg,

	ParamError: ParamErrorMsg,

	//AuthError:           LoginCheckErrorMsg,
	//UserRegisterFail:    UserRegisterFailMsg,
	//UserNameEmpty:       UserNameEmptyMsg,
	//UserPassEmpty:       UserPassEmptyMsg,
	//UserDisable:         UserDisableMsg,
	//Forbidden:           ForbiddenMsg,
	InternalServerError: InternalServerErrorMsg,

	CreateK8SClusterError: CreateK8SClusterErrorMsg,
	//
	//LDAPUserLoginFailed: LDAPUserLoginFailedMsg,
	//LDAPUserNotFound:    LDAPUserNotFoundMsg,
}

func ResultOk(code int, data interface{}, msg string, c *gin.Context) {

	c.JSON(http.StatusOK, Response{
		Code: code,
		Data: data,
		Msg:  msg,
	})
}

func ResultFail(code int, data interface{}, msg string, c *gin.Context) {

	if msg == "" {
		c.JSON(http.StatusOK, Response{
			Code:   code,
			Data:   data,
			ErrMsg: CustomError[code],
		})
	} else {
		c.JSON(http.StatusOK, Response{
			Code:   code,
			Data:   data,
			ErrMsg: msg,
		})
	}
}

const (
	SUCCESS = 0
	ERROR   = 7000

	ParamError = 8000

	//AuthError           = 1000
	//UserRegisterFail    = 1003
	//UserNameEmpty       = 1004
	//UserPassEmpty       = 1005
	//UserDisable         = 1006
	//Forbidden           = http.StatusForbidden
	InternalServerError = http.StatusInternalServerError

	CreateK8SClusterError = 2000
	//
	//LDAPUserLoginFailed = 3000
	//LDAPUserNotFound    = 3001
)

func Ok(c *gin.Context) {
	ResultOk(SUCCESS, map[string]interface{}{}, "操作成功", c)
}

func Fail(c *gin.Context) {
	ResultFail(ERROR, map[string]interface{}{}, "操作失败", c)
}

func OkWithData(data interface{}, c *gin.Context) {
	ResultOk(SUCCESS, data, "操作成功", c)
}

func FailWithMessage(code int, message string, c *gin.Context) {
	ResultFail(code, map[string]interface{}{}, message, c)
}
