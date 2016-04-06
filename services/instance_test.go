package services

import (
	"testing"

	"github.com/namely/broadway/instance"
	"github.com/namely/broadway/store"
	"github.com/stretchr/testify/assert"
)

func TestCreateInstance(t *testing.T) {
	nt := newNotificationTestHelper()
	defer nt.Close()
	store := store.New()
	service := NewInstanceService(store)

	i := &instance.Instance{PlaybookID: "test", ID: "222"}
	err := service.Create(i)
	assert.Nil(t, err)
	createdInstance, _ := service.Show(i.PlaybookID, i.ID)
	assert.Equal(t, "test", createdInstance.PlaybookID)
	assert.Equal(t, instance.StatusNew, createdInstance.Status)
	assert.Contains(t, nt.requestBody, "created")
}

func TestShow(t *testing.T) {
	store := store.New()
	service := NewInstanceService(store)

	i := &instance.Instance{PlaybookID: "test", ID: "222"}
	err := service.Create(i)
	i, err = service.Show(i.PlaybookID, i.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test", i.PlaybookID)
	assert.Equal(t, "222", i.ID)
}

func TestShowMissingInstance(t *testing.T) {
	store := store.New()
	service := NewInstanceService(store)

	i := &instance.Instance{PlaybookID: "test", ID: "broken"}
	i, err := service.Show(i.PlaybookID, i.ID)
	assert.NotNil(t, err)
	assert.Nil(t, i, "PlaybookID should be empty")
}

func TestAllWithPlaybookID(t *testing.T) {
	store := store.New()
	service := NewInstanceService(store)

	i := &instance.Instance{PlaybookID: "test", ID: "222"}
	err := service.Create(i)
	if err != nil {
		t.Log(err)
	}

	instances, err := service.AllWithPlaybookID(i.PlaybookID)
	assert.Nil(t, err)
	assert.NotEmpty(t, instances)
}
