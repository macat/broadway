package deployment

import (
	"os"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/restclient"
	coreclient "k8s.io/kubernetes/pkg/client/typed/generated/core/v1"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/runtime/serializer"

	// Install API
	_ "k8s.io/kubernetes/pkg/api/install"

	"github.com/namely/broadway/manifest"
	"github.com/namely/broadway/playbook"
)

var groupVersionKind = unversioned.GroupVersionKind{
	Group:   api.GroupName,
	Version: runtime.APIVersionInternal,
	Kind:    meta.AnyKind,
}

var client coreclient.CoreInterface
var deserializer runtime.Decoder
var namespace string

// Step represents a deployment step
type Step interface {
	Deploy() error
}

// TaskStep combines a task and a step
type TaskStep struct {
	task *playbook.Task
	step Step
}

func init() {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)
	factory := serializer.NewCodecFactory(scheme)
	deserializer = factory.UniversalDeserializer()

	kcfg := &restclient.Config{
		Host:     "http://localhost:8080",
		Insecure: true,
	}
	var err error
	client, err = coreclient.NewForConfig(kcfg)
	if err != nil {
		panic(err)
	}

	namespace = os.Getenv("KUBERNETES_NAMESPACE")
}

// Deployer declares something that can Deploy Deployments
type Deployer interface {
	Deploy(playbook.Playbook, map[string]string) error
}

// Deployment represents a deployment of an instance
type Deployment struct {
	Playbook  playbook.Playbook
	Variables map[string]string
	Manifests map[string]*manifest.Manifest
}

// Deploy executes the deployment
func (d *Deployment) Deploy() error {
	tasksteps, err := d.collectSteps()
	if err != nil {
		return err
	}

	for _, taskstep := range tasksteps {
		err := taskstep.step.Deploy()
		if err != nil {
			return err
		}
	}

	return nil
}

// collectSteps collects all the steps for the deployment from the playbook
func (d *Deployment) collectSteps() ([]TaskStep, error) {
	var steps = []TaskStep{}
	for _, task := range d.Playbook.Tasks {
		if task.PodManifest != "" {
			m := d.Manifests[task.PodManifest]
			rendered := m.Execute(d.Variables)
			object, err := deserialize(rendered)
			if err != nil {
				return steps, err
			}
			step := NewPodmanifestStep(object)
			steps = append(steps, TaskStep{&task, step})
		}
		for _, name := range task.Manifests {
			m := d.Manifests[name]
			rendered := m.Execute(d.Variables)
			object, err := deserialize(rendered)
			if err != nil {
				return steps, err
			}
			step := NewManifestStep(object)
			steps = append(steps, TaskStep{&task, step})
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
