package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type SecretManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newSecretManager(clusters *ClusterManager) *SecretManager {
	return &SecretManager{clusters: clusters}
}

func (m *SecretManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	secret := ctx.Object.(*types.Secret)
	if err := createSecret(cluster.KubeClient, namespace, secret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate secret name %s", secret.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create secret failed %s", err.Error()))
		}
	}

	secret.SetID(secret.Name)
	return secret, nil
}

func (m *SecretManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sSecrets, err := getSecrets(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list secret info failed:%s", err.Error())
		}
		return nil
	}

	var secrets []*types.Secret
	for _, secret := range k8sSecrets.Items {
		secrets = append(secrets, k8sSecretToSCSecret(&secret))
	}
	return secrets
}

func (m SecretManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	secret := ctx.Object.(*types.Secret)
	k8sSecret, err := getSecret(cluster.KubeClient, namespace, secret.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get secret info failed:%s", err.Error())
		}
		return nil
	}

	return k8sSecretToSCSecret(k8sSecret)
}

func (m SecretManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	secret := ctx.Object.(*types.Secret)
	err := deleteSecret(cluster.KubeClient, namespace, secret.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("secret %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete secret failed %s", err.Error()))
		}
	}
	return nil
}

func getSecret(cli client.Client, namespace, name string) (*corev1.Secret, error) {
	secret := corev1.Secret{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &secret)
	return &secret, err
}

func getSecrets(cli client.Client, namespace string) (*corev1.SecretList, error) {
	secrets := corev1.SecretList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &secrets)
	return &secrets, err
}

func createSecret(cli client.Client, namespace string, secret *types.Secret) error {
	data := make(map[string][]byte)
	for k, v := range secret.Data {
		data[k] = []byte(v)
	}

	k8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secret.Name, Namespace: namespace, Labels: map[string]string{"creator": "singlecloud"}},
		Data:       data,
		Type:       corev1.SecretTypeOpaque,
	}
	return cli.Create(context.TODO(), k8sSecret)
}

func deleteSecret(cli client.Client, namespace, name string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), secret)
}

func k8sSecretToSCSecret(k8sSecret *corev1.Secret) *types.Secret {
	data := make(map[string]string)
	for k, v := range k8sSecret.Data {
		data[k] = string(v)
	}

	secret := &types.Secret{
		Name: k8sSecret.Name,
		Data: data,
	}
	secret.SetID(k8sSecret.Name)
	secret.SetType(types.SecretType)
	secret.SetCreationTimestamp(k8sSecret.CreationTimestamp.Time)
	return secret
}
