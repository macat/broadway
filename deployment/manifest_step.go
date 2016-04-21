package deployment

import (
	"errors"

	"github.com/golang/glog"
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

// Deploy executes the deployment of a step
func (s *ManifestStep) Deploy() error {
	oGVK := s.object.GetObjectKind().GroupVersionKind()
	switch oGVK.Kind {
	case "ReplicationController":
		o := s.object.(*v1.ReplicationController)

		rc, err := client.ReplicationControllers(namespace).Get(o.ObjectMeta.Name)

		if err == nil && rc != nil {
			glog.Info("Updating replication controller: ", o.ObjectMeta.Name)
			_, err = client.ReplicationControllers(namespace).Update(o)
		} else {
			glog.Info("Creating new replication controller: ", o.ObjectMeta.Name)
			_, err = client.ReplicationControllers(namespace).Create(o)
		}
		if err != nil {
			glog.Info("Create or Update failed: ", err)
			return err
		}
	case "Pod":
		o := s.object.(*v1.Pod)
		_, err := client.Pods(namespace).Get(o.ObjectMeta.Name)

		if err != nil {
			glog.Info("Creating new pod: ", o.ObjectMeta.Name)
			_, err = client.Pods(namespace).Create(o)
		} else {
			glog.Info("Updating pod", o.ObjectMeta.Name)
			_, err = client.Pods(namespace).Update(o)
		}
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
