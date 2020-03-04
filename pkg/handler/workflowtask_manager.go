package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"

	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1alpha2 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha2"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	wft := ctx.Resource.(*types.WorkFlowTask)

	wf, err := getWorkFlow(cluster.GetKubeClient(), ns, ctx.Resource.GetParent().GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "workflow doesn't exist")
		}
		return nil, resterror.NewAPIError(resterror.ClusterUnavailable, fmt.Sprintf("get owner workflow failed %s", err.Error()))
	}

	pipelineRun, err := genPipelineRun(cluster.GetKubeClient(), ns, wf, wft)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ClusterUnavailable, fmt.Sprintf("gen workflowtask failed %s", err.Error()))
	}

	if err := cluster.GetKubeClient().Create(context.TODO(), pipelineRun); err != nil {
		return nil, resterror.NewAPIError(resterror.ClusterUnavailable, fmt.Sprintf("create k8s workflowtask failed %s", err.Error()))
	}

	wft.SetID(pipelineRun.Name)
	wft.SetCreationTimestamp(time.Now())
	return wft, nil
}

func (m *WorkFlowTaskManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	return nil
}

func (m *WorkFlowTaskManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}
	return nil
}

func (m *WorkFlowTaskManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}
	return nil
}

func genPipelineRun(cli client.Client, namespace string, wf *types.WorkFlow, wft *types.WorkFlowTask) (*tektonv1alpha1.PipelineRun, error) {
	tasks := genPipelineTasks(wf)

	yaml, err := getPipelineRunDeployYaml(cli, namespace, wf, wft)
	if err != nil {
		return nil, err
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
						StringVal: yaml},
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
								Name: "git-source",
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
			Name: "deploy",
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
	return getWorkFlowDeployYaml(cli, namespace, &deploy)
}
