package deployment

import (
	"errors"
	"reflect"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/runtime"
)

// ManifestStep implements a deployment step
type ManifestStep struct {
	object runtime.Object
}

var _ Step = &ManifestStep{}

// NewManifestStep creates a default step
func NewManifestStep(object runtime.Object) Step {
	return &ManifestStep{
		object: object,
	}
}

func compareContainers(a, b v1.Container) bool {
	return a.Name == b.Name && a.Image == b.Image &&
		reflect.DeepEqual(a.Command, b.Command) &&
		reflect.DeepEqual(a.Args, b.Args) && a.WorkingDir == b.WorkingDir &&
		reflect.DeepEqual(a.Ports, b.Ports) &&
		reflect.DeepEqual(a.Env, b.Env) &&
		reflect.DeepEqual(a.Resources, b.Resources) &&
		a.ImagePullPolicy == b.ImagePullPolicy
}

func compareContainerLists(a, b []v1.Container) bool {
	aMap := map[string]v1.Container{}
	bMap := map[string]v1.Container{}

	for _, c := range a {
		aMap[c.Name] = c
	}
	for _, c := range b {
		bMap[c.Name] = c
	}

	if len(aMap) != len(bMap) {
		return false
	}

	for name, ac := range aMap {
		bc, ok := bMap[name]
		if !ok {
			return false
		}
		if !compareContainers(ac, bc) {
			return false
		}
	}
	return true
}

func comparePodSpecs(a, b v1.PodSpec) bool {
	if len(a.Containers) != len(b.Containers) {
		return false
	}
	if !compareContainerLists(a.Containers, b.Containers) {
		return false
	}
	return reflect.DeepEqual(a.ImagePullSecrets, b.ImagePullSecrets)
}

func comparePods(a, b *v1.Pod) bool {
	return reflect.DeepEqual(a.ObjectMeta, b.ObjectMeta) &&
		comparePodSpecs(a.Spec, b.Spec)
}

func compareRCs(a, b *v1.ReplicationController) bool {
	if a.ObjectMeta.Name == "" {
		return false
	}
	return reflect.DeepEqual(a.ObjectMeta, b.ObjectMeta) &&
		reflect.DeepEqual(a.Spec.Replicas, b.Spec.Replicas) &&
		reflect.DeepEqual(a.Spec.Selector, b.Spec.Selector) &&
		reflect.DeepEqual(a.Spec.Template.ObjectMeta, b.Spec.Template.ObjectMeta) &&
		comparePodSpecs(a.Spec.Template.Spec, b.Spec.Template.Spec)
}

// Deploy executes the deployment of a step
func (s *ManifestStep) Deploy() error {
	oGVK := s.object.GetObjectKind().GroupVersionKind()
	switch oGVK.Kind {
	case "ReplicationController":
		var o *v1.ReplicationController
		switch s.object.(type) {
		case *v1.ReplicationController:
			o = s.object.(*v1.ReplicationController)
		case *api.ReplicationController:
			rr := s.object.(*api.ReplicationController)
			if err := scheme.Convert(rr, o); err != nil {
				glog.Error("API object conversion failed.")
				return err
			}
		}
		rc, err := client.ReplicationControllers(namespace).Get(o.ObjectMeta.Name)
		if err == nil && rc != nil {
			if compareRCs(rc, o) {
				glog.Info("Existing RC is identical, skipping deployment")
				return nil
			}

			glog.Info("Deleting old replication controller: ", o.ObjectMeta.Name)
			err = client.ReplicationControllers(namespace).Delete(o.ObjectMeta.Name, nil)

			for k := 1; err == nil && k < 20; k++ {
				time.Sleep(200 * time.Millisecond) // Wait for Kubernetes to delete the resource
				_, err = client.ReplicationControllers(namespace).Get(o.ObjectMeta.Name)
			}
			time.Sleep(2 * time.Second) // Wait for Kubernetes to delete pods
		}

		glog.Info("Creating new replication controller: ", o.ObjectMeta.Name)
		_, err = client.ReplicationControllers(namespace).Create(o)

		if err != nil {
			glog.Error("Create or Update failed: ", err)
			return err
		}
	case "Pod":
		o := s.object.(*v1.Pod)
		pod, err := client.Pods(namespace).Get(o.ObjectMeta.Name)

		if err == nil && pod != nil {
			if comparePods(pod, o) {
				glog.Info("Existing Pod is identical, skipping deployment")
				return nil
			}
			glog.Info("Deleting old pod", o.ObjectMeta.Name)
			err = client.Pods(namespace).Delete(o.ObjectMeta.Name, nil)

			for k := 1; err == nil && k < 20; k++ {
				time.Sleep(200 * time.Millisecond) // Wait for Kubernetes to delete the resource
				_, err = client.Pods(namespace).Get(o.ObjectMeta.Name)
			}
		}

		glog.Info("Creating new pod: ", o.ObjectMeta.Name)
		_, err = client.Pods(namespace).Create(o)
		if err != nil {
			glog.Info("Create or Update failed: ", err)
			return err
		}
	case "Service":
		o := s.object.(*v1.Service)
		service, err := client.Services(namespace).Get(o.ObjectMeta.Name)

		if err != nil {
			glog.Info("Creating new service: ", o.ObjectMeta.Name)
			_, err = client.Services(namespace).Create(o)
		} else {
			glog.Info("Updating service", o.ObjectMeta.Name)
			o.ObjectMeta.ResourceVersion = service.ObjectMeta.ResourceVersion
			o.Spec.ClusterIP = service.Spec.ClusterIP
			_, err = client.Services(namespace).Update(o)
		}
		if err != nil {
			glog.Info("Create or Update failed: ", err)
			return err
		}
	default:
		return errors.New("Kubernetes resource is not recognized: " + oGVK.Kind)
	}
	return nil
}

// Destroy deletes kubernetes resource
func (s *ManifestStep) Destroy() error {
	var err error
	oGVK := s.object.GetObjectKind().GroupVersionKind()
	meta, err := api.ObjectMetaFor(s.object)
	if err != nil {
		return err
	}
	switch oGVK.Kind {
	case "ReplicationController":
		err = client.ReplicationControllers(namespace).Delete(meta.Name, nil)
	case "Service":
		err = client.Services(namespace).Delete(meta.Name, nil)
	case "Pod":
		err = client.Pods(namespace).Delete(meta.Name, nil)
	}
	return err
}
