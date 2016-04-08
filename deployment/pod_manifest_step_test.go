package deployment

import (
	"testing"

	"errors"

	"github.com/stretchr/testify/assert"

	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/client/typed/generated/core/v1/fake"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
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
		Events   func(*watch.FakeWatcher, *v1.Pod)
	}{
		{
			Name:     "Simple Pod create",
			Object:   mustDeserialize(podt1),
			Expected: nil,
			Events: func(w *watch.FakeWatcher, pod *v1.Pod) {
				pod.Status = v1.PodStatus{
					Phase: v1.PodPending,
				}
				w.Modify(pod)
				pod.Status = v1.PodStatus{
					Phase: v1.PodRunning,
				}
				w.Modify(pod)
				pod.Status = v1.PodStatus{
					Phase: v1.PodSucceeded,
				}
				w.Modify(pod)
			},
		},
		{
			Name:     "Simple Pod failure",
			Object:   mustDeserialize(podt1),
			Expected: errors.New("Setup pod failed!"),
			Events: func(w *watch.FakeWatcher, pod *v1.Pod) {
				pod.Status = v1.PodStatus{
					Phase: v1.PodPending,
				}
				w.Modify(pod)
				pod.Status = v1.PodStatus{
					Phase: v1.PodRunning,
				}
				w.Modify(pod)
				pod.Status = v1.PodStatus{
					Phase: v1.PodFailed,
				}
				w.Modify(pod)
			},
		},
	}

	for _, c := range cases {
		// Reset client
		pod := mustDeserialize(podt1).(*v1.Pod)
		f := &core.Fake{}
		w := watch.NewFake()
		f.AddWatchReactor("*", core.DefaultWatchReactor(w, nil))

		go c.Events(w, pod)

		client = &fake.FakeCore{f}
		step := NewPodManifestStep(c.Object)
		client.(*fake.FakeCore).Fake.ClearActions()
		assert.Equal(t, 0, len(f.Actions()), c.Name+" action count did not reset")
		err := step.Deploy()
		assert.Equal(t, c.Expected, err, c.Name+" error was not expected result")
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
