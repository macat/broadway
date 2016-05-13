package deployment

import (
	"errors"
	"time"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
)

// PodManifestStep implements a deployment step with pod manifest
type PodManifestStep struct {
	object runtime.Object
}

var _ Step = &PodManifestStep{}

// NewPodManifestStep creates a PodManifestStep and returns a Step
func NewPodManifestStep(object runtime.Object) Step {
	return &PodManifestStep{
		object: object,
	}
}

// Deploy executes the deployment step
func (s *PodManifestStep) Deploy() error {
	var err error
	oGVK := s.object.GetObjectKind().GroupVersionKind()
	if oGVK.Kind != "Pod" {
		return errors.New("Incorrect Pod Manifest type: " + oGVK.Kind)
	}

	o := s.object.(*v1.Pod)
	p, err := client.Pods(namespace).Get(o.ObjectMeta.Name)
	if err != nil {
		glog.Warningln(err)
	}
	if p.ObjectMeta.Name == o.ObjectMeta.Name {
		glog.Infoln("Deleting old pod: ", o.ObjectMeta.Name)
		err = client.Pods(namespace).Delete(o.ObjectMeta.Name, nil)
		time.Sleep(1 * time.Second)
		if err != nil {
			glog.Errorln(err)
			return err
		}
	}

	glog.Infoln("Creating new pod: ", o.ObjectMeta.Name)
	_, err = client.Pods(namespace).Create(o)
	if err != nil {
		glog.Errorln("Creating Setup Pod Failed: ", err)
		return err
	}

	glog.Infoln("Watching setup pod...")
	selector := fields.Set{"metadata.name": o.ObjectMeta.Name}.AsSelector()
	lo := api.ListOptions{Watch: true, FieldSelector: selector}
	watcher, err := client.Pods(namespace).Watch(lo)
	defer watcher.Stop()
	var pod *v1.Pod

	for {
		var ok bool
		event := <-watcher.ResultChan()
		pod, ok = event.Object.(*v1.Pod)
		if !ok {
			podv := v1.Pod{}
			apipod := event.Object.(*api.Pod)
			err = api.Scheme.Convert(apipod, &podv)
			if err != nil {
				glog.Errorln("API Object conversion failed: ", err)
				return err
			}
			pod = &podv
		}

		if pod.Status.Phase != v1.PodPending && pod.Status.Phase != v1.PodRunning {
			glog.Infoln("NOT PENDING AND NOT RUNNING")
			break
		}
	}
	glog.Infoln("Setup pod finished: ", o.ObjectMeta.Name)
	if pod.Status.Phase == v1.PodFailed {
		glog.Errorln("Setup pod failed: ", o.ObjectMeta.Name)
		return errors.New("Setup pod failed!")
	}
	if pod.Status.Phase == v1.PodUnknown {
		glog.Errorln("Setup pod failed: ", o.ObjectMeta.Name)
		return errors.New("State of Pod Unknown")
	}
	err = client.Pods(namespace).Delete(o.ObjectMeta.Name, nil)
	if err != nil {
		return err
	}

	return nil
}

// Destroy deletes pod
func (s *PodManifestStep) Destroy() error {
	return nil
}
