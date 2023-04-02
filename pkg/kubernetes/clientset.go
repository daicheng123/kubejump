package kubernetes

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ClientSet struct {
	K8sClientSet    *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	DiscoveryClient *discovery.DiscoveryClient
	restConfig      *rest.Config
	clientErr       error
}

func (cs *ClientSet) initClientSet(masterUrl, caData, bearerToken string) error {
	cs.Config(masterUrl, caData, bearerToken)

	cs.K8sClientSet, cs.clientErr = kubernetes.NewForConfig(cs.restConfig)

	cs.DynamicClient, cs.clientErr = dynamic.NewForConfig(cs.restConfig)

	cs.DiscoveryClient, cs.clientErr = discovery.NewDiscoveryClientForConfig(cs.restConfig)

	return cs.clientErr
}

func (cs *ClientSet) Config(masterUrl, caData, bearerToken string) *rest.Config {
	tlsClientConfig := rest.TLSClientConfig{CAData: []byte(caData)}
	cs.restConfig = &rest.Config{
		Host:            masterUrl,
		TLSClientConfig: tlsClientConfig,
		BearerToken:     bearerToken,
		QPS:             100,
		Burst:           150,
	}

	return cs.restConfig
}
