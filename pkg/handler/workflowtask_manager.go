package handler

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"

	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1alpha2 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha2"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	k8stypes "k8s.io/apimachinery/pkg/types"
	knativeapis "knative.dev/pkg/apis"
)

const (
	zcloudWorkFlowBuildImage      = "zdnscloud/kaniko-executor:v0.13.0"
	zcloudWorkFlowYamlWriterImage = "zdnscloud/workflow-yaml-writer:v0.0.1"
	zcloudWorkFlowDeployerImage   = "zdnscloud/kubectl:v1.17.2"
)

type WorkFlowTaskManager struct {
	clusters *ClusterManager
}

func newWorkFlowTaskManager(clusters *ClusterManager) *WorkFlowTaskManager {
	return &WorkFlowTaskManager{
		clusters: clusters,
	}
}

func (m *WorkFlowTaskManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	ns := ctx.Resource.GetParent().GetParent().GetID()
	wfID := ctx.Resource.GetParent().GetID()
	wft := ctx.Resource.(*types.WorkFlowTask)

	return wft, createWorkFlowTask(cluster.GetKubeClient(), ns, wfID, wft)
}

func createWorkFlowTask(cli client.Client, namespace, wfID string, wft *types.WorkFlowTask) *resterror.APIError {
	wf, err := getWorkFlow(cli, namespace, wfID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, "workflow doesn't exist")
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get owner workflow failed %s", err.Error()))
	}

	tasks, err := getWorkFlowTasks(cli, namespace, wfID)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get workflowtasks failed %s", err.Error()))
	}

	if IsNoCompleteTaskExists(tasks) {
		return resterror.NewAPIError(resterror.PermissionDenied, "exist running or pending task")
	}

	pipelineRun, err := genPipelineRun(cli, namespace, wf, wft)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("gen workflowtask failed %s", err.Error()))
	}

	if err := cli.Create(context.TODO(), pipelineRun); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("create k8s workflowtask failed %s", err.Error()))
	}

	if err := updateWorkFlowLastestIDAnnotation(cli, namespace, wfID, pipelineRun.Name); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update workflow lastest task id annotation failed %s", err.Error()))
	}

	wft.SetID(pipelineRun.Name)
	wft.SetCreationTimestamp(time.Now())
	return nil
}

func updateWorkFlowLastestIDAnnotation(cli client.Client, namespace, workFlow, workFlowTaskID string) error {
	pr := tektonv1alpha1.PipelineResource{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, workFlow}, &pr); err != nil {
		return err
	}
	pr.Annotations[zcloudWorkFlowLatestTaskIDAnnotationKey] = workFlowTaskID
	return cli.Update(context.TODO(), &pr)
}

func IsNoCompleteTaskExists(ts []*types.WorkFlowTask) bool {
	for _, t := range ts {
		if t.Status.CurrentStatus == "Running" || t.Status.CurrentStatus == "Pending" {
			return true
		}
	}
	return false
}

func (m *WorkFlowTaskManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	ns := ctx.Resource.GetParent().GetParent().GetID()
	id := ctx.Resource.GetID()

	wft, err := getWorkFlowTask(cluster.GetKubeClient(), ns, id)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("workflowtask %s doesn't exist", id))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get namespace %s workflow %s failed %s", ns, id, err.Error()))
	}
	return wft, nil
}

func getWorkFlowTask(cli client.Client, namespace, name string) (*types.WorkFlowTask, error) {
	pr := tektonv1alpha1.PipelineRun{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &pr); err != nil {
		return nil, err
	}

	return k8sPipelineRunToWorkFlowTask(pr), nil
}

func k8sPipelineRunToWorkFlowTask(p tektonv1alpha1.PipelineRun) *types.WorkFlowTask {
	w := types.WorkFlowTask{
		SubTasks: k8sPipelineRunToWorkFlowSubTasks(p),
		Status:   k8sPipelineRunToWorkFlowTaskStatus(p),
	}
	for _, param := range p.Spec.Params {
		if param.Name == "IMAGE_TAG" {
			w.ImageTag = param.Value.StringVal
			break
		}
	}
	w.SetID(p.Name)
	w.SetCreationTimestamp(p.CreationTimestamp.Time)
	if p.DeletionTimestamp != nil {
		w.SetDeletionTimestamp(p.DeletionTimestamp.Time)
	}
	return &w
}

func k8sPipelineRunToWorkFlowTaskStatus(p tektonv1alpha1.PipelineRun) types.WorkFlowTaskStatus {
	s := types.WorkFlowTaskStatus{}
	if p.Status.StartTime != nil {
		s.StartTime = resource.ISOTime(p.Status.StartTime.Time)
	}
	if p.Status.CompletionTime != nil {
		s.CompletionTime = resource.ISOTime(p.Status.CompletionTime.Time)
	}

	condition := p.Status.GetCondition(knativeapis.ConditionSucceeded)
	if condition != nil {
		s.CurrentStatus = condition.Reason
		s.Message = condition.Message
	}
	return s
}

func k8sPipelineRunToWorkFlowSubTasks(p tektonv1alpha1.PipelineRun) []types.WorkFlowSubTask {
	tasks := []types.WorkFlowSubTask{}
	for _, pipelineTask := range p.Spec.PipelineSpec.Tasks {
		task := types.WorkFlowSubTask{Name: pipelineTask.Name}
		for _, v := range p.Status.TaskRuns {
			if v.PipelineTaskName == pipelineTask.Name {
				taskStatus := types.WorkFlowTaskStatus{}
				if v.Status.StartTime != nil {
					taskStatus.StartTime = resource.ISOTime(v.Status.StartTime.Time)
				}
				if v.Status.CompletionTime != nil {
					taskStatus.CompletionTime = resource.ISOTime(v.Status.CompletionTime.Time)
				}
				condition := v.Status.GetCondition(knativeapis.ConditionSucceeded)
				if condition != nil {
					taskStatus.CurrentStatus = condition.Reason
					taskStatus.Message = condition.Message
				}
				containers := []string{}
				if taskStatus.CurrentStatus != "Pending" {
					for _, step := range v.Status.Steps {
						containers = append(containers, step.ContainerName)
					}
				}
				task.Status = taskStatus
				task.PodName = v.Status.PodName
				task.Containers = containers
			}
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func (m *WorkFlowTaskManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	ns := ctx.Resource.GetParent().GetParent().GetID()
	wfID := ctx.Resource.GetParent().GetID()

	ts, err := getWorkFlowTasks(cluster.GetKubeClient(), ns, wfID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list workflow task of %s-%s failed %s", ns, wfID, err.Error()))
	}
	return ts, nil
}

func getWorkFlowTasks(cli client.Client, namespace, workFlowName string) ([]*types.WorkFlowTask, error) {
	ps, err := getPipelineRunsByWorkFlowName(cli, namespace, workFlowName)
	if err != nil {
		return nil, err
	}

	ts := types.WorkFlowTasks{}
	for _, p := range ps {
		ts = append(ts, k8sPipelineRunToWorkFlowTask(p))
	}

	sort.Sort(sort.Reverse(ts))
	return ts, nil
}

func deletePipelineRunsByWorkFlowName(cli client.Client, namespace, name string) error {
	ps, err := getPipelineRunsByWorkFlowName(cli, namespace, name)
	if err != nil {
		return err
	}

	for _, p := range ps {
		if err := cli.Delete(context.TODO(), &p); err != nil {
			return err
		}
	}
	return nil
}

func getPipelineRunsByWorkFlowName(cli client.Client, namespace, name string) ([]tektonv1alpha1.PipelineRun, error) {
	pl := tektonv1alpha1.PipelineRunList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{zcloudWorkFlowIDLabelKey: name}})
	if err != nil {
		return nil, err
	}
	listOptions := &client.ListOptions{Namespace: namespace, LabelSelector: selector}
	if err := cli.List(context.TODO(), listOptions, &pl); err != nil {
		return nil, err
	}
	return pl.Items, nil
}

func genPipelineRun(cli client.Client, namespace string, wf *types.WorkFlow, wft *types.WorkFlowTask) (*tektonv1alpha1.PipelineRun, error) {
	tasks := genPipelineTasks(wf)

	var deployYaml string
	if wf.AutoDeploy {
		yaml, err := getPipelineRunDeployYaml(cli, namespace, wf, wft)
		if err != nil {
			return nil, err
		}
		deployYaml = yaml
	}

	p := &tektonv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      genWorkFlowRandomName(wf.Name),
			Namespace: namespace,
			Labels: map[string]string{
				zcloudWorkFlowIDLabelKey: wf.Name,
			},
		},
		Spec: tektonv1alpha1.PipelineRunSpec{
			ServiceAccountName: wf.Name,
			Params: []tektonv1alpha2.Param{
				tektonv1alpha2.Param{
					Name: "IMAGE_URL",
					Value: tektonv1alpha2.ArrayOrString{
						Type:      tektonv1alpha2.ParamTypeString,
						StringVal: wf.Image.Name,
					},
				},
				tektonv1alpha2.Param{
					Name: "IMAGE_TAG",
					Value: tektonv1alpha2.ArrayOrString{
						Type:      tektonv1alpha2.ParamTypeString,
						StringVal: wft.ImageTag},
				},
				tektonv1alpha2.Param{
					Name: "DEPLOY_YAML",
					Value: tektonv1alpha2.ArrayOrString{
						Type:      tektonv1alpha2.ParamTypeString,
						StringVal: deployYaml},
				},
			},
			Resources: []tektonv1alpha1.PipelineResourceBinding{
				tektonv1alpha1.PipelineResourceBinding{
					Name:        "git-source",
					ResourceRef: &tektonv1alpha1.PipelineResourceRef{Name: wf.Name},
				},
			},
			PipelineSpec: &tektonv1alpha1.PipelineSpec{
				Params: []tektonv1alpha2.ParamSpec{
					tektonv1alpha2.ParamSpec{
						Name:        "IMAGE_URL",
						Type:        tektonv1alpha2.ParamTypeString,
						Description: "The Url of image repository",
					},
					tektonv1alpha2.ParamSpec{
						Name:        "IMAGE_TAG",
						Type:        tektonv1alpha2.ParamTypeString,
						Description: "The Tag to apply to the built image",
					},
					tektonv1alpha2.ParamSpec{
						Name:        "DEPLOY_YAML",
						Type:        tektonv1alpha2.ParamTypeString,
						Description: "The deployment yaml to deploy",
					},
				},
				Resources: []tektonv1alpha1.PipelineDeclaredResource{
					tektonv1alpha1.PipelineDeclaredResource{
						Name: "git-source",
						Type: tektonv1alpha1.PipelineResourceTypeGit,
					},
				},
				Tasks: tasks,
			},
		},
	}
	return p, nil
}

func genPipelineTasks(wf *types.WorkFlow) []tektonv1alpha1.PipelineTask {
	ts := []tektonv1alpha1.PipelineTask{
		tektonv1alpha1.PipelineTask{
			Name: "build",
			Params: []tektonv1alpha2.Param{
				tektonv1alpha2.Param{
					Name: "IMAGE_URL",
					Value: tektonv1alpha2.ArrayOrString{
						Type:      tektonv1alpha2.ParamTypeString,
						StringVal: "$(params.IMAGE_URL)",
					},
				},
				tektonv1alpha2.Param{
					Name: "IMAGE_TAG",
					Value: tektonv1alpha2.ArrayOrString{
						Type:      tektonv1alpha2.ParamTypeString,
						StringVal: "$(params.IMAGE_TAG)"},
				},
			},
			Resources: &tektonv1alpha1.PipelineTaskResources{
				Inputs: []tektonv1alpha1.PipelineTaskInputResource{
					tektonv1alpha1.PipelineTaskInputResource{
						Name:     "source",
						Resource: "git-source",
					},
				},
			},
			TaskSpec: &tektonv1alpha1.TaskSpec{
				Inputs: &tektonv1alpha1.Inputs{
					Params: []tektonv1alpha2.ParamSpec{
						tektonv1alpha2.ParamSpec{
							Name:        "IMAGE_URL",
							Type:        tektonv1alpha2.ParamTypeString,
							Description: "The Url of image repository",
						},
						tektonv1alpha2.ParamSpec{
							Name:        "IMAGE_TAG",
							Type:        tektonv1alpha2.ParamTypeString,
							Description: "The Tag to apply to the built image",
						},
						tektonv1alpha2.ParamSpec{
							Name:        "DOCKERFILE",
							Type:        tektonv1alpha2.ParamTypeString,
							Description: "Path to the Dockerfile to build",
							Default: &tektonv1alpha2.ArrayOrString{
								Type:      tektonv1alpha2.ParamTypeString,
								StringVal: "./Dockerfile"},
						},
						tektonv1alpha2.ParamSpec{
							Name:        "CONTEXT",
							Type:        tektonv1alpha2.ParamTypeString,
							Description: "The build context used by Kaniko",
							Default: &tektonv1alpha2.ArrayOrString{
								Type:      tektonv1alpha2.ParamTypeString,
								StringVal: "./"},
						},
					},
					Resources: []tektonv1alpha1.TaskResource{
						tektonv1alpha1.TaskResource{
							tektonv1alpha1.ResourceDeclaration{
								Name: "source",
								Type: tektonv1alpha1.PipelineResourceTypeGit,
							},
						},
					},
				},
				Steps: []tektonv1alpha1.Step{
					tektonv1alpha1.Step{
						Container: corev1.Container{
							Name:       "build-and-push",
							WorkingDir: "/workspace/source",
							Image:      zcloudWorkFlowBuildImage,
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name:  "DOCKER_CONFIG",
									Value: "/tekton/home/.docker",
								},
							},
							Command: []string{
								"/kaniko/executor",
								"--dockerfile=$(inputs.params.DOCKERFILE)",
								"--context=/workspace/source/$(inputs.params.CONTEXT)",
								"--destination=$(inputs.params.IMAGE_URL):$(inputs.params.IMAGE_TAG)",
								"--skip-tls-verify",
							},
						},
					},
				},
			},
		},
	}

	if wf.AutoDeploy {
		deployTask := tektonv1alpha1.PipelineTask{
			Name:     "deploy",
			RunAfter: []string{"build"},
			Params: []tektonv1alpha2.Param{
				tektonv1alpha2.Param{
					Name: "DEPLOY_YAML",
					Value: tektonv1alpha2.ArrayOrString{
						Type:      tektonv1alpha2.ParamTypeString,
						StringVal: "$(params.DEPLOY_YAML)"},
				},
			},
			TaskSpec: &tektonv1alpha1.TaskSpec{
				Inputs: &tektonv1alpha1.Inputs{
					Params: []tektonv1alpha2.ParamSpec{
						tektonv1alpha2.ParamSpec{
							Name:        "DEPLOY_YAML",
							Type:        tektonv1alpha2.ParamTypeString,
							Description: "The deployment yaml to deploy",
						},
					},
				},
				Steps: []tektonv1alpha1.Step{
					tektonv1alpha1.Step{
						Container: corev1.Container{
							Name:       "write-yaml",
							WorkingDir: "/workspace/source",
							Image:      zcloudWorkFlowYamlWriterImage,
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name:  "DEPLOY_YAML",
									Value: "$(inputs.params.DEPLOY_YAML)",
								},
							},
							Command: []string{"writer"},
							Args: []string{
								"-env",
								"DEPLOY_YAML",
								"-out",
								"/workspace/deploy.yaml",
							},
						},
					},
					tektonv1alpha1.Step{
						Container: corev1.Container{
							Name:       "deploy-yaml",
							WorkingDir: "/workspace/source",
							Image:      zcloudWorkFlowDeployerImage,
							Command:    []string{"kubectl"},
							Args: []string{
								"apply",
								"-f",
								"/workspace/deploy.yaml",
							},
						},
					},
				},
			},
		}
		ts = append(ts, deployTask)
	}
	return ts
}

func getPipelineRunDeployYaml(cli client.Client, namespace string, wf *types.WorkFlow, wft *types.WorkFlowTask) (string, error) {
	if !wf.AutoDeploy {
		return "", nil
	}
	deploy := wf.Deploy
	containers := []types.Container{}
	if len(wf.Deploy.Containers) > 0 {
		for _, container := range wf.Deploy.Containers {
			if strings.Contains(container.Image, wf.Image.Name) {
				container.Image = fmt.Sprintf("%s:%s", wf.Image.Name, wft.ImageTag)
			}
			containers = append(containers, container)
		}
	}
	deploy.Containers = containers
	return k8sDeployToYaml(cli, namespace, &deploy)
}

func k8sDeployToYaml(cli client.Client, namespace string, deploy *types.Deployment) (string, error) {
	k8sDeploy, pvcs, err := scDeployToK8sDeployAndPvcs(cli, namespace, deploy)
	if err != nil {
		return "", err
	}

	serializer := k8sJson.NewSerializerWithOptions(
		k8sJson.DefaultMetaFactory, nil, nil,
		k8sJson.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)

	out := bytes.NewBuffer(make([]byte, 0, 64))
	if err := serializer.Encode(k8sDeploy, out); err != nil {
		return "", err
	}
	out.WriteString("---\n")
	for _, pv := range pvcs {
		if err := serializer.Encode(&pv, out); err != nil {
			return "", err
		}
		out.WriteString("---\n")
	}
	return out.String(), nil
}

func scDeployToK8sDeployAndPvcs(cli client.Client, namespace string, deploy *types.Deployment) (*appsv1.Deployment, []corev1.PersistentVolumeClaim, error) {
	podTemplate, k8sPVCs, err := getWorkLoadPodTempateSpecAndPvcs(namespace, deploy, cli)
	if err != nil {
		return nil, nil, err
	}
	replicas := int32(deploy.Replicas)
	k8sDeploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: generatePodOwnerObjectMeta(namespace, deploy),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deploy.Name},
			},
			Template: *podTemplate,
		},
	}

	for i := range k8sPVCs {
		k8sPVCs[i].TypeMeta = metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		}
	}
	return k8sDeploy, k8sPVCs, nil
}

func getWorkLoadPodTempateSpecAndPvcs(namespace string, podOwner interface{}, cli client.Client) (*corev1.PodTemplateSpec, []corev1.PersistentVolumeClaim, error) {
	structVal := reflect.ValueOf(podOwner).Elem()
	advancedOpts := structVal.FieldByName("AdvancedOptions").Interface().(types.AdvancedOptions)
	containers := structVal.FieldByName("Containers").Interface().([]types.Container)
	pvs := structVal.FieldByName("PersistentVolumes").Interface().([]types.PersistentVolumeTemplate)

	k8sPodSpec, k8sPVCs, err := scPodSpecToK8sPodSpecAndPVCs(containers, pvs)
	if err != nil {
		return nil, nil, err
	}

	name := structVal.FieldByName("Name").String()
	meta, err := createPodTempateObjectMeta(name, namespace, cli, advancedOpts, containers)
	if err != nil {
		return nil, nil, err
	}

	return &corev1.PodTemplateSpec{
		ObjectMeta: meta,
		Spec:       k8sPodSpec,
	}, k8sPVCs, nil
}
