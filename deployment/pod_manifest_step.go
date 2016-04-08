package deployment

import (
	"errors"
	"log"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/conversion"
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
		log.Println(err)
	}
	log.Println(p)
	if p.ObjectMeta.Name == o.ObjectMeta.Name {
		log.Println("Deleting old pod: ", o.ObjectMeta.Name)
		err = client.Pods(namespace).Delete(o.ObjectMeta.Name, nil)
		if err != nil {
			return err
		}
	}

	log.Println("Creating new pod: ", o.ObjectMeta.Name)
	_, err = client.Pods(namespace).Create(o)
	if err != nil {
		log.Println("Creating Setup Pod Failed: ", err)
		return err
	}

	log.Println("Watching setup pod...")
	selector := fields.Set{"metadata.name": o.ObjectMeta.Name}.AsSelector()
	lo := api.ListOptions{Watch: true, FieldSelector: selector}
	watcher, err := client.Pods(namespace).Watch(lo)
	defer watcher.Stop()
	var pod *v1.Pod

	for {
		event := <-watcher.ResultChan()
		log.Printf("%T", event)
		err = DefaultConvert(event.Object.(*api.Pod), pod, flags, meta)
		if err != nil {
			log.Println("conversion ", err)
		}
		// switch event.(type) {
		// case *v1.:
		pod = event.Object.(*v1.Pod)
		// case *api.Pod:
		// 	v1.Convert_api_Pod_To_v1_Pod(event.Object.(*api.Pod), pod)
		// }

		if pod.Status.Phase != v1.PodPending && pod.Status.Phase != v1.PodRunning {
			log.Println("NOT PENDING AND NOT RUNNING")
			break
		}
	}

	log.Println("Setup pod finished: ", o.ObjectMeta.Name)
	if pod.Status.Phase == v1.PodFailed {
		log.Println("Setup pod failed: ", o.ObjectMeta.Name)
		return errors.New("Setup pod failed!")
	}
	if pod.Status.Phase == v1.PodUnknown {
		log.Println("Setup pod failed: ", o.ObjectMeta.Name)
		return errors.New("State of Pod Unknown")
	}
	err = client.Pods(namespace).Delete(o.ObjectMeta.Name, nil)
	if err != nil {
		return err
	}

	return nil
}
