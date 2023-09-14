package proxy

import (
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
)

type ConnectionOption func(options *ConnectionOptions)

func ConnectContainer(info *ContainerInfo) ConnectionOption {
	return func(opts *ConnectionOptions) {
		opts.k8sContainer = info
	}
}

func ConnectTokenAuthInfo(authInfo *entity.ConnectInfo) ConnectionOption {
	return func(opts *ConnectionOptions) {
		opts.authInfo = authInfo
	}
}

type ConnectionOptions struct {
	authInfo     *entity.ConnectInfo
	k8sContainer *ContainerInfo
}

type ContainerInfo struct {
	CLuster   *entity.ClusterConfig
	Namespace string
	PodName   string
	Container string
}

func (c *ContainerInfo) String() string {
	return fmt.Sprintf("%s_%s_%s_%s", c.CLuster.ClientUniqKey(), c.Namespace, c.PodName, c.Container)
}
