package cloudcredentials

import (
	"os"

	coreV1 "github.com/rancher/rancher/pkg/generated/norman/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var DOAccessToken string = os.Getenv("DO_ACCESSKEY")

type CloudCredential struct {
	coreV1.Interface
}

// NewCloudCredentialSecret is a method that creates a *v1.Secret specific to a cloud credential type
func NewCloudCredentialSecret(cloudCredentialName, description, driverType, namespace string) *v1.Secret {
	data := make(map[string][]byte)

	switch driverType {
	case "digitalocean":
		data = map[string][]byte{
			"accessToken": []byte(DOAccessToken),
		}
	}
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudCredentialName,
			Namespace: namespace,
			Annotations: map[string]string{
				"field.cattle.io/description":   description,
				"provisioning.cattle.io/driver": driverType,
			},
		},

		Data: data,
		Type: "provisioning.cattle.io/cloud-credential",
	}
}

// NewCloudCredential creates a CloudCredential object
func NewCloudCredential(client coreV1.Interface) *CloudCredential {
	return &CloudCredential{
		client,
	}
}

// CreateCloudCredential is function that creates a cloud credential using a CloudCredential object with a specified v1.Secret
func (c *CloudCredential) CreateCloudCredential(secret *v1.Secret) (*v1.Secret, error) {
	returnedSecret, err := c.Secrets(secret.Namespace).Create(secret)
	return returnedSecret, err
}

// UpdateCloudCredential is function that updates a cloud credential using a CloudCredential object
func (c *CloudCredential) UpdateCloudCredential(secret *v1.Secret) (*v1.Secret, error) {
	returnedSecret, err := c.Secrets(secret.Namespace).Update(secret)
	return returnedSecret, err
}

// DeleteCloudCredential is function that deletes a cloud credential using a CloudCredential object
func (c *CloudCredential) DeleteCloudCredential(secret *v1.Secret) error {
	return c.Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
}

// GetCloudCredential is function that gets a cloud credential using a CloudCredential object
func (c *CloudCredential) GetCloudCredential(secret *v1.Secret) (*v1.Secret, error) {
	returnedSecret, err := c.Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})
	return returnedSecret, err
}

// ListCloudCredential is function that lists cloud credentials from a specific namespace using a CloudCredential object
func (c *CloudCredential) ListCloudCredential(nameSpace string) (*v1.SecretList, error) {
	returnedSecret, err := c.Secrets(nameSpace).List(metav1.ListOptions{})
	return returnedSecret, err
}
