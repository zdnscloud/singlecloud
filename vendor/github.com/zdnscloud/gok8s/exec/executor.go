package exec

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/predicate"
)

var (
	errCmdTimeout = errors.New("pod isn't ready, cmd timeout")
)

var (
	executorLanchedPodMark = "zcloud-executor"
)

type ResizeableStream interface {
	io.ReadWriter
	remotecommand.TerminalSizeQueue
}

type Executor struct {
	k8sCfg     *rest.Config
	client     client.Client
	podWatcher *podWatcher
	stopCh     chan struct{}
}

func New(k8sCfg *rest.Config, client client.Client, cache cache.Cache) (*Executor, error) {
	stopCh := make(chan struct{})
	ctrl := controller.New("podWatcher", cache, scheme.Scheme)
	ctrl.Watch(&corev1.Pod{})
	podWatcher := newPodWatcher()
	go ctrl.Start(stopCh, podWatcher, predicate.NewIgnoreUnchangedUpdate())

	return &Executor{
		k8sCfg:     k8sCfg,
		client:     client,
		podWatcher: podWatcher,
		stopCh:     stopCh,
	}, nil
}

type Pod struct {
	Namespace          string
	Name               string
	Container          string
	Image              string
	ServiceAccountName string
}

type Cmd struct {
	Path string
	Args []string
}

func (e *Executor) Stop() {
	close(e.stopCh)
}

func (e *Executor) CreatePod(p Pod, c Cmd, timeout time.Duration) error {
	if _, err := e.createPod(p, c); err != nil {
		return err
	}

	return e.waitPodReady(p, timeout)
}

func (e *Executor) Exec(p Pod, c Cmd, rw ResizeableStream) error {
	clientset, err := kubernetes.NewForConfig(e.k8sCfg)
	if err != nil {
		return err
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(p.Name).
		Namespace(p.Namespace).
		SubResource("exec").
		Param("container", p.Container).
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("command", c.Path).
		Param("tty", "true")

	req.VersionedParams(
		&corev1.PodExecOptions{
			Container: p.Name,
			Command:   []string{},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		},
		scheme.ParameterCodec,
	)
	executor, err := remotecommand.NewSPDYExecutor(
		e.k8sCfg, http.MethodPost, req.URL(),
	)
	if err != nil {
		return err
	} else {
		return executor.Stream(remotecommand.StreamOptions{
			Stdin:             rw,
			Stdout:            rw,
			Stderr:            rw,
			Tty:               true,
			TerminalSizeQueue: rw,
		})
	}
}

func (e *Executor) waitPodReady(p Pod, timeout time.Duration) error {
	ready := e.podWatcher.AddNotifyTask(p.Namespace, p.Name)
	select {
	case <-ready:
		return nil
	case <-time.After(timeout):
		return errCmdTimeout
	}
}

var (
	privileged               = false
	defaultUser              = int64(1000)
	defaultGroup             = int64(2000)
	allowPrivilegeEscalation = false
)

func (e *Executor) createPod(p Pod, c Cmd) (*corev1.Pod, error) {
	termPeroidSeonds := int64(0)
	kp := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			Labels:    map[string]string{"app": executorLanchedPodMark},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName:            p.ServiceAccountName,
			TerminationGracePeriodSeconds: &termPeroidSeonds,
			RestartPolicy:                 corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					TTY:   false,
					Stdin: false,
					Name:  p.Container,
					Image: p.Image,
					SecurityContext: &corev1.SecurityContext{
						Privileged:               &privileged,
						RunAsUser:                &defaultUser,
						RunAsGroup:               &defaultGroup,
						AllowPrivilegeEscalation: &allowPrivilegeEscalation,
					},
					ImagePullPolicy: corev1.PullPolicy(corev1.PullAlways),
				},
			},
		},
	}

	err := e.client.Create(context.TODO(), kp)
	if apierrors.IsAlreadyExists(err) {
		err = nil
	}
	return kp, err
}
