package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/client/typed/generated/core/v1/fake"
	"k8s.io/kubernetes/pkg/runtime"
)

func init() {
	namespace = "test"
}

func mustDeserialize(manifest string) runtime.Object {
	o, err := deserialize(manifest)
	if err != nil {
		panic(err)
	}
	return o
}

func TestManifestStepDeploy(t *testing.T) {
	cases := []struct {
		Name     string
		Object   runtime.Object
		Expected []string
		Before   func(*core.Fake)
	}{
		{
			Name:     "Simple RC create",
			Object:   mustDeserialize(rct1),
			Expected: []string{"get", "delete", "create"},
			Before: func(f *core.Fake) {
				rc := mustDeserialize(rct3).(*v1.ReplicationController)
				o := core.NewObjects(api.Scheme, api.Codecs.UniversalDecoder())
				if err := o.Add(rc); err != nil {
					panic(err)
				}
				f.AddReactor("*", "*", core.ObjectReaction(o, api.RESTMapper))
			},
		},
		{
			Name:     "RC identical update",
			Object:   mustDeserialize(rct1),
			Expected: []string{"get"},
			Before: func(f *core.Fake) {
				rc := mustDeserialize(rct1).(*v1.ReplicationController)

				o := core.NewObjects(api.Scheme, api.Codecs.UniversalDecoder())
				if err := o.Add(rc); err != nil {
					panic(err)
				}

				f.AddReactor("*", "*", core.ObjectReaction(o, api.RESTMapper))
			},
		},
		{
			Name:     "RC simple update",
			Object:   mustDeserialize(rct1),
			Expected: []string{"get", "delete", "create"},
			Before: func(f *core.Fake) {
				rc := mustDeserialize(rct2).(*v1.ReplicationController)

				o := core.NewObjects(api.Scheme, api.Codecs.UniversalDecoder())
				if err := o.Add(rc); err != nil {
					panic(err)
				}

				f.AddReactor("*", "*", core.ObjectReaction(o, api.RESTMapper))
			},
		},
	}

	for _, c := range cases {
		// Reset client
		client = &fake.FakeCore{&core.Fake{}}
		f := client.(*fake.FakeCore).Fake
		step := NewManifestStep(c.Object)
		f.ReactionChain = nil
		c.Before(f)
		f.ClearActions()
		assert.Equal(t, 0, len(f.Actions()), c.Name+" action count did not reset")
		err := step.Deploy()
		assert.Nil(t, err, c.Name+" deploy returned with nil")

		verbs := map[string]bool{}
		for _, a := range f.Actions() {
			verbs[a.GetVerb()] = true
		}

		fired := map[string]bool{}
		for verb := range verbs {
			assert.Contains(t, c.Expected, verb, c.Name+" didn't expect this action.")
			fired[verb] = true
		}

		expected := map[string]bool{}
		for f := range fired {
			expected[f] = true
		}

		assert.Equal(t, expected, fired, c.Name+" actions don't match the actually fired actions.")
	}
}

func TestManifestStepDestroy(t *testing.T) {
	cases := []struct {
		Name     string
		Object   runtime.Object
		Expected string
		Before   func()
	}{
		{
			Name:     "Simple RC delete",
			Object:   mustDeserialize(rct1),
			Expected: "delete",
			Before:   func() {},
		},
	}

	for _, c := range cases {
		// Reset client
		client = &fake.FakeCore{&core.Fake{}}
		f := client.(*fake.FakeCore).Fake
		step := NewManifestStep(c.Object)
		c.Before()
		client.(*fake.FakeCore).Fake.ClearActions()
		assert.Equal(t, 0, len(f.Actions()), c.Name+" action count did not reset")
		err := step.Destroy()
		assert.Nil(t, err, c.Name+" deploy returned with nil")

		// manifest step should always fire only 1 actions
		assert.Equal(t, 1, len(f.Actions()), c.Name+" fired less/more than 2 actions")

		verbs := []string{}
		for _, a := range f.Actions() {
			verbs = append(verbs, a.GetVerb())
		}

		assert.Contains(t, verbs, c.Expected, c.Name+" actions didn't contain the expected verb")
	}
}

var rct1 = `apiVersion: v1
kind: ReplicationController
metadata:
  name: test2
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

var rct2 = `apiVersion: v1
kind: ReplicationController
metadata:
  name: test2
spec:
  replicas: 1
  selector:
    name: redis-2
  template:
    metadata:
      labels:
        name: redis
    spec:
      containers:
      - name: redis
        image: kubernetes/redis:v1
`

var rct3 = `apiVersion: v1
kind: ReplicationController
metadata:
  name: test3
spec:
  replicas: 1
  selector:
    name: redis-2
  template:
    metadata:
      labels:
        name: redis
    spec:
      containers:
      - name: redis
        image: kubernetes/redis:v1
`
