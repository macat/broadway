package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/client/typed/generated/core/v1/fake"
)

func init() {
	client = &fake.FakeCore{&core.Fake{}}
}

func TestDeploy(t *testing.T) {
	cases := []struct {
		Name     string
		Tasks    []Task
		Expected int
	}{
		{
			Name: "Step with one manifest file",
			Tasks: []Task{
				{
					Name: "First step",
					Manifests: []string{
						"test",
					},
				},
			},
			Expected: 2,
		}, {
			Name: "Step with 3 manifest files",
			Tasks: []Task{
				{
					Name: "First step",
					Manifests: []string{
						"test",
						"test2",
						"test2",
					},
				},
			},
			Expected: 6,
		},
	}

	vars := map[string]string{
		"test": "ok",
	}
	m, _ := NewManifest("test", mtemplate)
	manifests := map[string]*Manifest{
		"test":  m,
		"test2": m,
	}

	for _, c := range cases {
		// Reset client
		client.(*fake.FakeCore).Fake.ClearActions()

		p := &Playbook{
			ID:    "test",
			Name:  "Test deployment",
			Meta:  Meta{},
			Vars:  []string{"test"},
			Tasks: c.Tasks,
		}

		d := &KubernetesDeployment{
			Playbook:  p,
			Variables: vars,
			Manifests: manifests,
		}

		err := d.Deploy()
		assert.Nil(t, err, c.Name+" deployment should not return with error")
	}
}

func TestDestroy(t *testing.T) {
	cases := []struct {
		Name     string
		Tasks    []Task
		Expected int
	}{
		{
			Name: "Step with one manifest file",
			Tasks: []Task{
				{
					Name: "First step",
					Manifests: []string{
						"test",
					},
				},
			},
			Expected: 1,
		}, {
			Name: "Step with 3 manifest files",
			Tasks: []Task{
				{
					Name: "First step",
					Manifests: []string{
						"test",
						"test2",
						"test2",
					},
				},
			},
			Expected: 3,
		},
	}

	vars := map[string]string{
		"test": "ok",
	}
	m, _ := NewManifest("test", mtemplate)
	manifests := map[string]*Manifest{
		"test":  m,
		"test2": m,
	}

	for _, c := range cases {
		// Reset client
		client.(*fake.FakeCore).Fake.ClearActions()

		p := &Playbook{
			ID:    "test",
			Name:  "Test deployment",
			Meta:  Meta{},
			Vars:  []string{"test"},
			Tasks: c.Tasks,
		}

		d := &KubernetesDeployment{
			Playbook:  p,
			Variables: vars,
			Manifests: manifests,
		}

		err := d.Destroy()
		assert.Nil(t, err, c.Name+" deployment should not return with error")
		f := client.(*fake.FakeCore).Fake
		assert.Equal(t, c.Expected, len(f.Actions()), c.Name+" should trigger actions.")
	}
}

var mtemplate = `apiVersion: v1
kind: ReplicationController
metadata:
  name: test
spec:
  replicas: 1
  selector:
    name: redis
  template:
    metadata:
      labels:
        name: redis
    spec:
      containers:
      - name: redis
        image: kubernetes/redis:v1
`
