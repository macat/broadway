package deployment

import (
	"github.com/golang/glog"
	"github.com/namely/broadway/env"

	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/restclient"
	coreclient "k8s.io/kubernetes/pkg/client/typed/generated/core/v1"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/runtime/serializer"

	// Install API
	_ "k8s.io/kubernetes/pkg/api/install"
)

var groupVersionKind = unversioned.GroupVersionKind{
	Group:   v1.GroupName,
	Version: runtime.APIVersionInternal,
	Kind:    meta.AnyKind,
}

var client coreclient.CoreInterface
var deserializer runtime.Decoder
var namespace string
var scheme *runtime.Scheme

// Step represents a deployment step
type Step interface {
	Deploy() error
}

// TaskStep combines a task and a step
type TaskStep struct {
	task *Task
	step Step
}

func init() {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)
	factory := serializer.NewCodecFactory(scheme)
	deserializer = factory.UniversalDeserializer()

	namespace = env.K8sNamespace
}

// KubernetesDeployment represents a deployment of an instance
type KubernetesDeployment struct {
	Playbook  *Playbook
	Variables map[string]string
	Manifests map[string]*Manifest
}

// NewKubernetesDeployment creates a new kuberentes deployment
func NewKubernetesDeployment(config *restclient.Config, playbook *Playbook, variables map[string]string, manifests map[string]*Manifest) (*KubernetesDeployment, error) {
	var err error
	client, err = coreclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &KubernetesDeployment{
		Playbook:  playbook,
		Variables: variables,
		Manifests: manifests,
	}, nil
}

// Deploy executes the deployment
func (d *KubernetesDeployment) Deploy() error {
	tasksteps, err := d.steps()
	if err != nil {
		return err
	}

	for i, taskstep := range tasksteps {
		glog.Infof("%d. Deploying Task %s...", i, taskstep.task.Name)
		err := taskstep.step.Deploy()
		if err != nil {
			glog.Warning("%d. step failed.")
			return err
		}
		glog.Infof("Done.")
	}

	return nil
}

func (d *KubernetesDeployment) steps() ([]TaskStep, error) {
	var steps = []TaskStep{}
	for _, task := range d.Playbook.Tasks {
		if task.PodManifest != "" {
			m := d.Manifests[task.PodManifest]
			rendered := m.Execute(d.Variables)
			object, err := deserialize(rendered)
			if err != nil {
				glog.Warningf("Failed to parse pod manifest %s - %s", task.Name, task.PodManifest)
				return steps, err
			}
			step := NewPodManifestStep(object)
			steps = append(steps, TaskStep{&task, step})
		} else {
			for _, name := range task.Manifests {
				m := d.Manifests[name]
				rendered := m.Execute(d.Variables)
				object, err := deserialize(rendered)
				if err != nil {
					glog.Warningf("Failed to parse manifest %s - %s", task.Name, name)
					return steps, err
				}
				step := NewManifestStep(object)
				steps = append(steps, TaskStep{&task, step})
			}
		}
	}
	return steps, nil
}

func deserialize(manifest string) (runtime.Object, error) {
	object, _, err := deserializer.Decode([]byte(manifest), &groupVersionKind, nil)
	if err != nil {
		return nil, err
	}
	return object, nil
}
