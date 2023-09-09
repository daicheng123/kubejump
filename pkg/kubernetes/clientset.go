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

const certData = `-----BEGIN CERTIFICATE-----
MIIDITCCAgmgAwIBAgIIZaqS+GgXj3QwDQYJKoZIhvcNAQELBQAwFTETMBEGA1UE
AxMKa3ViZXJuZXRlczAeFw0yMzA4MjgxMTE2NDdaFw0yNDA4MjcxMTE2NDlaMDQx
FzAVBgNVBAoTDnN5c3RlbTptYXN0ZXJzMRkwFwYDVQQDExBrdWJlcm5ldGVzLWFk
bWluMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAz6bzIKrDDhVMRGgu
LMA6VbMp5Gq3DwLgDGFkU41PL69gJfDTCd1r3b4drFfvGiaz+u5SqJ+3TOQZh4s8
RDdvHSg0XP6fKsRKn2owmFoom/DDn78uYVToA57N0uA+kOnt33F79TV0FDPfOsN3
sbIWbgkeApVdxFswfaYNHKMl9IP0VkrUjwRy9Ql1gD1XBJQ1mh1e8nJeXGczRXDU
3g3OvY9thT05JkqU19LZz8MgXQRiV/mFtuYsbCfOmIvcdXREWy1tokKgZBeyp8tD
lWC8g/plmIE3CJXPfWSmvFiLDuAmJn+GJJztz6zk2T5RJfvf8y2ysgM0HmI/UWya
UVVPiwIDAQABo1YwVDAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYIKwYBBQUH
AwIwDAYDVR0TAQH/BAIwADAfBgNVHSMEGDAWgBR2pxRfpFSOs6Dt4ZQN67JDemaK
HjANBgkqhkiG9w0BAQsFAAOCAQEAUv+8H/BSB7lpEEiUyH5fmaZy72CGBR9mKfyA
zjrEN9UDqvhRtggCczd/YqJT1/e3KpO3RBRHUbqbH/hXeXF+D/38BueWm0iYtUTK
gro1qBb5tbvhtYRaFP4D3wjObA+/r6ZarwI/csLh7+UoeDUaBJb5h9D6TcKcSQg7
Uc344xxcfN0nzoQ0Qi+ZpTmABF4PKwENUWK3EUYuSxd8yO7dNj0ScNYtqzQzGUT8
bhTMu/igPpZJ/MGB4SCAI1OJXr8MzUelvGGDpl99dQ5hEPXnztEzDzlpI1vSBkM8
GO74QwZzOTvqEor5Bw4yjJTZi+nRyjQt0EmoIrviNHJZH4PIig==
-----END CERTIFICATE-----`

const keyData = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAz6bzIKrDDhVMRGguLMA6VbMp5Gq3DwLgDGFkU41PL69gJfDT
Cd1r3b4drFfvGiaz+u5SqJ+3TOQZh4s8RDdvHSg0XP6fKsRKn2owmFoom/DDn78u
YVToA57N0uA+kOnt33F79TV0FDPfOsN3sbIWbgkeApVdxFswfaYNHKMl9IP0VkrU
jwRy9Ql1gD1XBJQ1mh1e8nJeXGczRXDU3g3OvY9thT05JkqU19LZz8MgXQRiV/mF
tuYsbCfOmIvcdXREWy1tokKgZBeyp8tDlWC8g/plmIE3CJXPfWSmvFiLDuAmJn+G
JJztz6zk2T5RJfvf8y2ysgM0HmI/UWyaUVVPiwIDAQABAoIBAQCWTLT2FCOS8f9+
FBo38ftHRKMx3bwadW5OB8BiaYnDbiEd1S4rmcUVfyJjOFKsjw7+tbnGq7Q1R3Tj
EvjQN3+Jjyw3k2UJw4Jv2KDL5ZY3KRGvcuXTNW2qESvRUtZ2dZvje3TJi6M1bEZL
dmgQimKJyreaDxsLoSV8DNC4xa4XSI1vEoOb2lbw6K9Ug3EzSnILq9g/Pwzn6I2M
vozNrD8DH66ci/Zwh3w2x8FLDszihIVQZ5yBeHQ4pjmvCGK4f1vvL0qh8SAV/Swl
CyrKfZXWNyDCNP47sAoZTntrz6X1VPvdH7sg/9PbMZppeNFPa2kS8HkLrUbSAQFM
tuQ11MGhAoGBAN2nWRsHSHXP8SL/RiEQO4f0ZQVXtVHfsIuN2cO4NP7glgAHBqKA
bmxZkMBT8cU3AcNslzEHYrGOl5Lsaz2ixqMZlu1C5rbbEO3ANjaHB8tXjJ5peODS
ymWMhz2R5oJ5Vzk38+PUPYaaYuFEjn/GRXPFkp8YstVJ4eLAv2wJ51CvAoGBAO/U
Lpts2Rg4Em2Aq4oFYWRpsrdckl7Y3EX8YyRWY++1FrC2sUXS1j1E7GCTdH06KjZT
V43SDC5geamACR/SbfeyxcnppYNN2YhtcmIZjI4Gj195AG79AjEjZZlg9SSvYVQb
MIOXfF5GxEYjchXuzwz3haQJXAOQcEh1WKmqks3lAoGAFuyJ4Ku+KMEa1V3FaQH+
xi7Wi9joXdFetvAyx3UztfCQUuxnGUNjKD2TJPEJnjX0Lrv6Xw2+fVKcBowBA1zk
YlXxMBStO9goRg6NDNKmUbd6SZ/q6oWifSItkoaWaoQWK0rIJJX8zwEEnPu0KS7e
W/zhDydEx55eiE8a/ReBHu0CgYAnG8Op5rsUHviqUKQocq4qEK5rBjZ8LwLkir6k
C05qpW4YzQHlb/ctsJKXQRRq41RE3ZxWbR09ZtAQGufh/4+dJ9LnFSm/Wq+Rdr+D
TcVE178Dg5jVgH6eArarp0rye4L0kyZ7HvXR7dpN0bPl7bZn1+k8EaamkiQtPK2D
pWQhHQKBgFQyLfFRgnzNkMQTK5ISjtqtLvp9g8Ir7yZWZqiwV3QWEawp+VneXvQ4
04nRT19q0vKB35Y8Bsvx/7BHw1rAxI298qR9UquBwpchCscj56BKvFAPgunh7J7l
5WMxzmhCainAw3zgCENexv78/Nl+JQayYd+A2OrobJ/4xJPHhl5W
-----END RSA PRIVATE KEY-----`

func (cs *ClientSet) Config(masterUrl, caData, bearerToken string) *rest.Config {
	tlsClientConfig := rest.TLSClientConfig{
		CAData:   []byte(caData),
		CertData: []byte(certData),
		KeyData:  []byte(keyData)}
	cs.restConfig = &rest.Config{
		Host:            masterUrl,
		TLSClientConfig: tlsClientConfig,
		//BearerToken:     bearerToken,
		QPS:   100,
		Burst: 150,
	}
	cs.restConfig.Insecure = false

	return cs.restConfig
}

func (cs *ClientSet) GetConfig() *rest.Config {
	return cs.restConfig
}
