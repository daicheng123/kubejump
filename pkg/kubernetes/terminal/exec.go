package terminal

import (
	"context"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/kubernetes"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"net/http"
)

// StartProcess is called by handleAttach
// Executed cmd in the container specified connects it up with the ptyHandler (a session)
func StartProcess(ctx context.Context, kcfg *entity.ClusterConfig, cmd []string, ptyHandler PtyHandler, namespace, podName, containerName string) error {
	factory, err := kubernetes.GetClientFactory()
	if err != nil {
		return err
	}
	k8sClient, err := factory.GetOrCreateClient(kcfg)
	if err != nil {
		return err
	}

	if len(containerName) < 1 {
		pod, err := k8sClient.K8sClientSet.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		containerName = pod.Spec.Containers[0].Name
	}
	req := k8sClient.K8sClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	cfg := k8sClient.Config(kcfg.MasterUrl, kcfg.CaData, kcfg.BearerToken)

	exec, err := remotecommand.NewSPDYExecutor(cfg, http.MethodPost, req.URL())
	if err != nil {
		return err
	}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             ptyHandler,
		Stdout:            ptyHandler,
		Stderr:            ptyHandler,
		TerminalSizeQueue: ptyHandler,
		Tty:               true,
	})
	if err != nil {
		return err
	}

	return nil
}
