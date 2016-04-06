package deployment

import (
	"testing"

	"errors"
	"github.com/stretchr/testify/assert"

	"k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/client/typed/generated/core/v1/fake"
	"k8s.io/kubernetes/pkg/runtime"
)

func init() {
	namespace = "test"
}

// func mustDeseralize(manifest string) runtime.Object {
// 	o, err := deserialize(manifest)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return o
// }

func TestPodManifestStepDeploy(t *testing.T) {
	cases := []struct {
		Name     string
		Object   runtime.Object
		Expected error
		Before   func()
	}{
		{
			Name:     "Simple Pod create",
			Object:   mustDeseralize(podt1),
			Expected: nil,
			Before:   func() {},
		},
		{
			Name:     "Simple Pod failure",
			Object:   mustDeseralize(podt1),
			Expected: errors.New("Setup pod failed!"),
			Before: func() {
				// rc := mustDeseralize(rct1).(*v1.ReplicationController)
				// client.ReplicationControllers("test").Create(rc)
			},
		},
	}

	for _, c := range cases {
		// Reset client
		client = &fake.FakeCore{&core.Fake{}}
		f := client.(*fake.FakeCore).Fake
		step := NewPodManifestStep(c.Object)
		c.Before()
		client.(*fake.FakeCore).Fake.ClearActions()
		assert.Equal(t, 0, len(f.Actions()), c.Name+" action count did not reset")
		err := step.Deploy()
		assert.Equal(t, c.Expected, err, c.Name+" error was not expected result")

		// manifest step should always fire only 2 actions
		// assert.Equal(t, 2, len(f.Actions()), c.Name+" fired less/more than 2 actions")

		// verbs := []string{}
		// for _, a := range f.Actions() {
		// 	verbs = append(verbs, a.GetVerb())
		// }
		//
		// assert.Contains(t, verbs, c.Expected, c.Name+" actions didn't contain the expected verb")
	}
}

var podt1 = `apiVersion: v1
kind: Pod
metadata:
  name: red
spec:
  containers:
  - name: redis
    image: kubernetes/redis:v1
`
