package handler

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	ErrDuplicateKeyInSecret = errors.New("duplicate key in secret")
)

type SecretManager struct {
	api.DefaultHandler
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

func (m *SecretManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	secret := ctx.Object.(*types.Secret)
	if err := updateSecret(cluster.KubeClient, namespace, secret); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update secret failed %s", err.Error()))
	} else {
		return secret, nil
	}
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
			log.Warnf("list secret info failed:%s", err.Error())
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
			log.Warnf("get secret info failed:%s", err.Error())
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
	k8sSecret, err := scSecretToK8sSecret(secret, namespace)
	if err != nil {
		return err
	} else {
		return cli.Create(context.TODO(), k8sSecret)
	}
}

func updateSecret(cli client.Client, namespace string, secret *types.Secret) error {
	k8sSecret, err := scSecretToK8sSecret(secret, namespace)
	if err != nil {
		return err
	} else {
		return cli.Update(context.TODO(), k8sSecret)
	}
}

func deleteSecret(cli client.Client, namespace, name string) error {
	k8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), k8sSecret)
}

func scSecretToK8sSecret(secret *types.Secret, namespace string) (*corev1.Secret, error) {
	data := make(map[string][]byte)
	for k, v := range secret.Data {
		if _, ok := data[k]; ok {
			return nil, ErrDuplicateKeyInSecret
		}
		data[k] = []byte(v)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secret.Name, Namespace: namespace},
		Data:       data,
		Type:       corev1.SecretTypeOpaque,
	}, nil
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
