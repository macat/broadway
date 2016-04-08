package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/namely/broadway/instance"
	"github.com/namely/broadway/playbook"
	"github.com/namely/broadway/store"
)

func TestDeployment(t *testing.T) {
	manifests, err := NewManifestService("../examples/manifests/").LoadManifestFolder()
	if err != nil {
		panic(err)
	}

	playbooks, err := playbook.LoadPlaybookFolder("../examples/playbooks")
	if err != nil {
		panic(err)
	}

	service := NewDeploymentService(store.New(), playbooks, manifests)

	cases := []struct {
		Name     string
		Instance *instance.Instance
		Error    error
		Expected instance.Status
	}{
		{
			Name: "Good Playbook",
			Instance: &instance.Instance{
				PlaybookID: "hello",
				ID:         "test",
				Vars: map[string]string{
					"version": "test",
				},
			},
			Error:    nil,
			Expected: instance.StatusDeployed,
		},
	}

	// i := &instance.Instance{
	// 	PlaybookID: "hello",
	// 	ID:         "test",
	// 	Vars: map[string]string{
	// 		"version": "test",
	// 	},
	// }

	for _, c := range cases {
		err = service.Deploy(c.Instance)
		assert.Equal(t, c.Error, err)
		assert.EqualValues(t, c.Expected, c.Instance.Status)
	}
}
