package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/namely/broadway/instance"
	"github.com/namely/broadway/playbook"
	"github.com/namely/broadway/store"
)

func TestDeployment(t *testing.T) {
	nt := newNotificationTestHelper()
	defer nt.Close()
	manifests, err := NewManifestService("../examples/manifests/").LoadManifestFolder()
	if err != nil {
		panic(err)
	}

	playbooks, err := playbook.LoadPlaybookFolder("../examples/playbooks")
	if err != nil {
		panic(err)
	}

	service := NewDeploymentService(store.New(), playbooks, manifests)

	i := &instance.Instance{
		PlaybookID: "hello",
		ID:         "test",
		Vars: map[string]string{
			"version": "test",
		},
	}

	err = service.Deploy(i)
	assert.Nil(t, err)
	assert.EqualValues(t, instance.StatusDeployed, i.Status)
	assert.Contains(t, nt.requestBody, "deployed")
}
