package deployment

import (
	"errors"
	"log"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
)

// PodmanifestStep implements a deployment step with pod manifest
type PodManifestStep struct {
	object runtime.Object
}

var _ Step = &PodManifestStep{}

// NewPodmanifestStep creates a podmanifest step and returns a Step
func NewPodManifestStep(object runtime.Object) Step {
	return &PodManifestStep{
		object: object,
	}
}

// Deploy executes the deployment step
func (s *PodManifestStep) Deploy() error {
	oGVK := s.object.GetObjectKind().GroupVersionKind()
	if oGVK.Kind != "Pod" {
		return errors.New("Incorrect Pod Manifest type: " + oGVK.Kind)
	}

	o := s.object.(*v1.Pod)
	_, err := client.Pods(namespace).Get(o.ObjectMeta.Name)
	if err != nil {
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
			log.Println(event)
			pod, err = client.Pods(namespace).Get(o.ObjectMeta.Name)
			if err != nil {
				return err
			}

			log.Println(event)
			log.Println(pod.Status.Message)
			log.Println("===")
			if pod.Status.Phase != "Pending" && pod.Status.Phase != "Running" {
				log.Println("NOT PENDING AND NOT RUNNING")
				break
			}
		}
		log.Println("Setup pod finished: ", o.ObjectMeta.Name)
		if pod.Status.Phase == "Failed" {
			log.Println("Setup pod failed: ", o.ObjectMeta.Name)
			return errors.New("Setup pod failed!")
		}
	}

	return nil
}
