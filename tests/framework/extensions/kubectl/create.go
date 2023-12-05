package kubectl

import (
	"context"
	"fmt"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	"github.com/rancher/rancher/tests/framework/extensions/unstructured"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1Unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func CreateUnstructured(s *session.Session, client *rancher.Client, content []byte, clusterID, n string, gvr schema.GroupVersionResource) (*v1Unstructured.Unstructured, error) {
	dynClient, _, err := setupDynamicClient(s, client, nil, clusterID)
	if err != nil {
		return nil, err
	}

	obj, _, err := v1Unstructured.UnstructuredJSONScheme.Decode(content, nil, nil)
	if err != nil {
		logrus.Fatal(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
		return nil, err
	}

	result, err := dynClient.Resource(gvr).Namespace(n).Create(context.TODO(), unstructured.MustToUnstructured(obj), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return result, nil
}
