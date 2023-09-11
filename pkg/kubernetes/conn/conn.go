package conn

import "errors"

var (
	InValidToken = errors.New("invalid token")
)

func IsValidK8sUserToken() bool {

}
