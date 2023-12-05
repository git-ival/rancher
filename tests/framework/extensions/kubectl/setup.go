package kubectl

import (
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	"github.com/rancher/rancher/tests/framework/extensions/kubeconfig"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/rancher/rancher/tests/framework/clients/dynamic"
)

func setupDynamicClient(s *session.Session, client *rancher.Client, scheme *runtime.Scheme, clusterID string) (*dynamic.Client, *session.Session, error) {
	kubeConfig, err := kubeconfig.GetKubeconfig(client, clusterID)
	if err != nil {
		return nil, s, err
	}

	restConfig, err := (*kubeConfig).ClientConfig()
	if err != nil {
		return nil, s, err
	}

	if scheme != nil {
		restConfig.ContentConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	}

	var session *session.Session
	if s == nil {
		session = client.Session.NewSession()
	} else {
		session = s
	}

	dynClient, err := dynamic.NewForConfig(session, restConfig)

	return dynClient, session, err
}
