package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/zdnscloud/cement/randomdata"
	"github.com/zdnscloud/gok8s/client"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	dockerhubRegistryURL     = "https://index.docker.io"
	dockerConfigJsonTemplate = `{"auths":{"%s":{"username":"%s","password":"%s","auth":"%s"}}}`
)

func genWorkFlowGitSecret(namespace string, wf *types.WorkFlow) *corev1.Secret {
	if wf.Git.User == "" || wf.Git.Password == "" {
		return nil
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      genWorkFlowRandomName(wf.Name),
			Namespace: namespace,
			Labels: map[string]string{
				zcloudWorkFlowIDLabelKey: wf.Name,
			},
		},
		Type: corev1.SecretTypeBasicAuth,
		StringData: map[string]string{
			"username": wf.Git.User,
			"password": wf.Git.Password,
		},
	}
}

func updateWorkFlowGitSecret(cli client.Client, secret *corev1.Secret, wf *types.WorkFlow) error {
	if wf.Git.User == "" || wf.Git.Password == "" {
		return nil
	}
	secret.StringData = map[string]string{
		"username": wf.Git.User,
		"password": wf.Git.Password,
	}
	return cli.Update(context.TODO(), secret)
}

func genWorkFlowDockerSecret(namespace string, wf *types.WorkFlow) *corev1.Secret {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", wf.Image.RegistryUser, wf.Image.RegistryPassword)))
	configJson := fmt.Sprintf(dockerConfigJsonTemplate, getWorkFlowDockerRegistryURL(wf.Image.Name), wf.Image.RegistryUser, wf.Image.RegistryPassword, auth)
	data := make(map[string][]byte)
	data[".dockerconfigjson"] = []byte(configJson)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      genWorkFlowRandomName(wf.Name),
			Namespace: namespace,
			Labels: map[string]string{
				zcloudWorkFlowIDLabelKey: wf.Name,
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: data,
	}
}

func updateWorkFlowDockerSecret(cli client.Client, secret *corev1.Secret, wf *types.WorkFlow) error {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", wf.Image.RegistryUser, wf.Image.RegistryPassword)))
	configJson := fmt.Sprintf(dockerConfigJsonTemplate, getWorkFlowDockerRegistryURL(wf.Image.Name), wf.Image.RegistryUser, wf.Image.RegistryPassword, auth)
	data := make(map[string][]byte)
	data[".dockerconfigjson"] = []byte(configJson)
	secret.Data = data
	return cli.Update(context.TODO(), secret)
}

func getWorkFlowDockerRegistryURL(image string) string {
	n := strings.Split(image, "/")
	if len(n) == 1 {
		return dockerhubRegistryURL
	}
	if strings.Contains(n[0], "docker.io") {
		return dockerhubRegistryURL
	}
	if len(strings.Split(n[0], ".")) > 1 {
		return fmt.Sprintf("https://%s", n[0])
	}
	return dockerhubRegistryURL
}

func genWorkFlowRandomName(workFlowName string) string {
	return fmt.Sprintf("%s-%s", workFlowName, randomdata.RandString(12))
}

func deleteWorkFlowSecrets(cli client.Client, namespace, name string) error {
	secrets, err := getWorkFlowSecrets(cli, namespace, name)
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if err := cli.Delete(context.TODO(), &secret); err != nil {
			return err
		}
	}
	return nil
}

func updateWorkFlowSecrets(cli client.Client, namespace string, wf *types.WorkFlow) error {
	secrets, err := getWorkFlowSecrets(cli, namespace, wf.Name)
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if secret.Type == corev1.SecretTypeDockerConfigJson {
			if err := updateWorkFlowDockerSecret(cli, &secret, wf); err != nil {
				return err
			}
		}
		if secret.Type == corev1.SecretTypeBasicAuth {
			if err := updateWorkFlowGitSecret(cli, &secret, wf); err != nil {
				return err
			}
		}
	}
	if len(secrets) == 1 && (wf.Git.User != "" && wf.Git.Password != "") {
		gitSecret := genWorkFlowGitSecret(namespace, wf)
		if err := cli.Create(context.TODO(), gitSecret); err != nil {
			return err
		}
	}
	return nil
}

func getWorkFlowSecrets(cli client.Client, namespace, name string) ([]corev1.Secret, error) {
	secrets := corev1.SecretList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{zcloudWorkFlowIDLabelKey: name}})
	if err != nil {
		return nil, err
	}
	listOptions := &client.ListOptions{Namespace: namespace, LabelSelector: selector}
	if err := cli.List(context.TODO(), listOptions, &secrets); err != nil {
		return nil, err
	}
	return secrets.Items, nil
}

func genWorkFlowServiceAccount(name, namespace, gitSecret, dockerSecret string) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace},
		Secrets: []corev1.ObjectReference{
			corev1.ObjectReference{Name: dockerSecret},
		},
	}

	if gitSecret != "" {
		sa.Secrets = append(sa.Secrets, corev1.ObjectReference{Name: gitSecret})
	}
	return sa
}

func addWorkFlowSaToCRB(cli client.Client, saName, saNamespace string) error {
	crb := &rbacv1.ClusterRoleBinding{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{Name: zcloudWorkFlowClusterRoleBindingName}, crb); err != nil {
		return err
	}
	crb.Subjects = append(crb.Subjects, rbacv1.Subject{
		Kind:      rbacv1.ServiceAccountKind,
		Name:      saName,
		Namespace: saNamespace,
	})
	return cli.Update(context.TODO(), crb)
}

func deleteWorkFlowSaFromCRB(cli client.Client, saName, saNamespace string) error {
	crb := &rbacv1.ClusterRoleBinding{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{Name: zcloudWorkFlowClusterRoleBindingName}, crb); err != nil {
		return err
	}

	subjects := []rbacv1.Subject{}
	for _, subject := range crb.Subjects {
		if subject.Name == saName && subject.Namespace == saNamespace {
			continue
		}
		subjects = append(subjects, subject)
	}
	crb.Subjects = subjects
	return cli.Update(context.TODO(), crb)
}

func genGitPipelineResource(cli client.Client, namespace string, wf *types.WorkFlow) (*tektonv1.PipelineResource, error) {
	wfJson, err := json.Marshal(wf)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow failed when gen git pipelineresource %s", err.Error())
	}

	r := &tektonv1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wf.Name,
			Namespace: namespace,
			Annotations: map[string]string{
				zcloudWorkFlowContentAnnotationKey: string(wfJson),
			}},
		Spec: tektonv1.PipelineResourceSpec{
			Type: tektonv1.PipelineResourceTypeGit,
			Params: []tektonv1.ResourceParam{
				tektonv1.ResourceParam{
					Name:  "url",
					Value: wf.Git.RepositoryURL,
				},
				tektonv1.ResourceParam{
					Name:  "revision",
					Value: wf.Git.Revision,
				},
			},
		},
	}
	return r, nil
}

func updateGitPipelineResource(cli client.Client, namespace string, wf *types.WorkFlow) error {
	wfJson, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("marshal workflow failed when update git pipelineresource %s", err.Error())
	}
	pr := tektonv1.PipelineResource{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, wf.Name}, &pr); err != nil {
		return err
	}

	pr.ObjectMeta.Annotations = map[string]string{
		zcloudWorkFlowContentAnnotationKey: string(wfJson),
	}
	pr.Spec.Params = []tektonv1.ResourceParam{
		tektonv1.ResourceParam{
			Name:  "url",
			Value: wf.Git.RepositoryURL,
		},
		tektonv1.ResourceParam{
			Name:  "revision",
			Value: wf.Git.Revision,
		},
	}
	return cli.Update(context.TODO(), &pr)
}

func deletePipelineResource(cli client.Client, namespace, name string) error {
	p := &tektonv1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), p)
}

func deleteWorkFlowDeploymentAndPVCs(cli client.Client, namespace, name string) error {
	k8sDeploy, err := getDeployment(cli, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	scDeploy, err := k8sDeployToSCDeploy(cli, k8sDeploy)
	if err != nil {
		return err
	}

	if err := deleteDeployment(cli, namespace, name); err != nil {
		return err
	}

	if delete, ok := k8sDeploy.Annotations[AnnkeyForDeletePVsWhenDeleteWorkload]; ok && delete == "true" {
		deleteWorkLoadPVCs(cli, namespace, k8sDeploy.Spec.Template.Spec.Volumes)
	}
	eb.PublishResourceDeleteEvent(scDeploy)
	return nil
}
