package services

import (
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/namely/broadway/deployment"
	"github.com/namely/broadway/instance"
	"github.com/namely/broadway/manifest"
	"github.com/namely/broadway/notification"
	"github.com/namely/broadway/playbook"
	"github.com/namely/broadway/store"
)

// DeploymentService implements the Broadway logic for deployments
type DeploymentService struct {
	repo      instance.Repository
	playbooks map[string]*playbook.Playbook
	manifests map[string]*manifest.Manifest
}

// NewDeploymentService creates a new DeploymentService
func NewDeploymentService(s store.Store, ps map[string]*playbook.Playbook, ms map[string]*manifest.Manifest) *DeploymentService {
	return &DeploymentService{
		repo:      instance.NewRepo(s),
		playbooks: ps,
		manifests: ms,
	}
}

func vars(i *instance.Instance) map[string]string {
	vs := map[string]string{}
	for k, v := range i.Vars {
		vs[k] = v
	}

	vs["playbook_id"] = i.PlaybookID
	vs["instance_id"] = i.ID
	vs["id"] = i.ID
	vs["instance_status"] = string(i.Status)

	return vs
}

// Deploy deploys a playbook
func (d *DeploymentService) Deploy(i *instance.Instance) error {
	playbook, ok := d.playbooks[i.PlaybookID]
	if !ok {
		return fmt.Errorf("Could not find playbook ID %s while deploying %s\n", i.PlaybookID, i.ID)
	}

	config, err := deployment.Config()
	if err != nil {
		return err
	}

	deployer, err := deployment.NewKubernetesDeployment(config, playbook, vars(i), d.manifests)
	if err != nil {
		return err
	}

	if i.Status == instance.StatusDeploying {
		return errors.New("Instance is being deployed already.")
	}

	if i.Status == instance.StatusDeleting {
		return errors.New("Instance is being deleted already.")
	}

	i.Status = instance.StatusDeploying
	err = d.repo.Save(i)
	if err != nil {
		glog.Errorf("Failed to save instance status Deploying for %s/%s, continuing deployment. Error: %s\n", i.PlaybookID, i.ID, err.Error())
	}

	err = deployer.Deploy()
	if err != nil {
		glog.Errorf("Deploying %s/%s failed: %s\n", i.PlaybookID, i.ID, err.Error())
		i.Status = instance.StatusError

		errS := d.repo.Save(i)
		if errS != nil {
			glog.Errorf("Failed to save instance status Error for %s/%s error: %s\n", i.PlaybookID, i.ID, errS.Error())
			return errS
		}
		return err
	}

	i.Status = instance.StatusDeployed
	err = d.repo.Save(i)
	if err != nil {
		glog.Errorf("Failed to save instance status Deployed for playbook ID %s, instance %s\n%s\n", i.PlaybookID, i.ID, err.Error())
		return err
	}

	err = sendDeploymentNotification(i)
	if err != nil {
		return err
	}

	return nil

}

func sendDeploymentNotification(i *instance.Instance) error {
	m := &notification.Message{
		Attachments: []notification.Attachment{
			{
				Text: fmt.Sprintf("Instance was deployed: %s %s.", i.PlaybookID, i.ID),
			},
		},
	}

	return m.Send()
}
