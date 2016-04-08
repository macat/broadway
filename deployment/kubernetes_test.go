package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/client/typed/generated/core/v1/fake"

	"github.com/namely/broadway/manifest"
	"github.com/namely/broadway/playbook"
)

func init() {
	client = &fake.FakeCore{&core.Fake{}}
}

func TestDeploy(t *testing.T) {
	cases := []struct {
		Name     string
		Tasks    []playbook.Task
		Expected int
	}{
		{
			Name: "Step with one manifest file",
			Tasks: []playbook.Task{
				{
					Name: "First step",
					Manifests: []string{
						"test",
					},
				},
			},
			Expected: 2,
		},
	}

	vars := map[string]string{
		"test": "ok",
	}
	m, _ := manifest.New("test", mtemplate)
	p, _ := manifest.New("test", ptemplate)
	manifests := map[string]*manifest.Manifest{
		"test":  m,
		"test2": m,
		"test3": p,
	}

	for _, c := range cases {
		// Reset client
		client.(*fake.FakeCore).Fake.ClearActions()

		p := &playbook.Playbook{
			ID:    "test",
			Name:  "Test deployment",
			Meta:  playbook.Meta{},
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
var ptemplate = `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
      containers:
      - name: redis
        image: kubernetes/redis:v1
`
