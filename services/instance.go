package services

import (
	"bytes"
	"fmt"
	"regexp"
	"text/template"
	"time"

	"github.com/golang/glog"
	"github.com/namely/broadway/deployment"
	"github.com/namely/broadway/instance"
	"github.com/namely/broadway/notification"
	"github.com/namely/broadway/store"
)

var sanitizer = regexp.MustCompile(`[^a-zA-Z0-9\-]`)
var validator = regexp.MustCompile(`^[a-zA-Z0-9\-]{1,253}$`)

// InstanceService definition
type InstanceService struct {
	repo instance.Repository
}

// NewInstanceService creates a new instance service
func NewInstanceService(s store.Store) *InstanceService {
	return &InstanceService{repo: instance.NewRepo(s)}
}

// PlaybookNotFound indicates a problem due to Broadway not knowing about a
// playbook
type PlaybookNotFound struct {
	playbookID string
}

func (e *PlaybookNotFound) Error() string {
	return fmt.Sprintf("Can't make instance because playbook %s is missing\n", e.playbookID)
}

// InvalidVar indicates a problem setting or updating an instance var that is not declared in that instance's playbook
type InvalidVar struct {
	playbookID string
	key        string
}

func (e *InvalidVar) Error() string {
	return fmt.Sprintf("Playbook %s does not declare a var named %s\n", e.playbookID, e.key)
}

// InvalidID indicates an id that does not match the format of a subdomain
type InvalidID struct {
	badID       string
	suggestedID string
}

func (e *InvalidID) Error() string {
	return fmt.Sprintf("%s is an invalid id; valid characters are dash and alphanumerics. Try %s", e.badID, e.suggestedID)
}

func validateID(id string) error {
	match := validator.FindStringIndex(id)
	if match == nil {
		x := sanitizer.ReplaceAllString(id, "-")
		if len(x) > 253 {
			x = x[0:253]
		}
		return &InvalidID{
			badID:       id,
			suggestedID: x,
		}
	}
	return nil
}

// CreateOrUpdate a new instance
func (is *InstanceService) CreateOrUpdate(i *instance.Instance) (*instance.Instance, error) {
	if err := validateID(i.ID); err != nil {
		return nil, err
	}

	existing, err := is.repo.FindByID(i.PlaybookID, i.ID)
	if err != nil {
		switch err.(type) {
		case instance.MalformedSavedData:
			return nil, err
		}
		i.Created = time.Now().Unix()
	} else {
		i.Status = existing.Status
	}

	pb, ok := deployment.AllPlaybooks[i.PlaybookID]
	if !ok {
		return nil, &PlaybookNotFound{i.PlaybookID}
	}

	// Create lookup map for playbok variables
	vs := make(map[string]bool)
	for _, v := range pb.Vars {
		vs[v] = true
	}

	// Validate vars
	for k := range i.Vars {
		if vs[k] != true {
			return nil, &InvalidVar{i.PlaybookID, k}
		}
	}

	vars := make(map[string]string)
	for _, pv := range pb.Vars {
		vars[pv] = "" // default to empty string
		if existing != nil {
			v, ok := existing.Vars[pv]
			if ok {
				vars[pv] = v // use existing value
			}
		}
		v, ok := i.Vars[pv]
		if ok == true {
			vars[pv] = v
		}
	}
	i.Vars = vars

	err = is.repo.Save(i)
	if err != nil {
		return nil, err
	}

	err = sendNotification(existing != nil, i)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// Update an instance
func (is *InstanceService) Update(i *instance.Instance) (*instance.Instance, error) {
	glog.Info("Instance Service: Update")
	err := is.repo.Save(i)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// Show takes playbookID and instanceID and returns the matching Instance, if
// any
func (is *InstanceService) Show(playbookID, ID string) (*instance.Instance, error) {
	instance, err := is.repo.FindByID(playbookID, ID)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

// AllWithPlaybookID returns all the instances for an specified playbook id
func (is *InstanceService) AllWithPlaybookID(playbookID string) ([]*instance.Instance, error) {
	return is.repo.FindByPlaybookID(playbookID)
}

// Delete removes an instance
func (is *InstanceService) Delete(i *instance.Instance) error {
	ii, err := is.Show(i.PlaybookID, i.ID)
	if err != nil {
		return err
	}

	return is.repo.Delete(ii)
}

func sendNotification(update bool, i *instance.Instance) error {
	pb, ok := deployment.AllPlaybooks[i.PlaybookID]
	if !ok {
		return fmt.Errorf("Failed to lookup playbook for instance %+v", *i)
	}

	s := "created"
	if update == true {
		s = "updated"
	}

	atts := []notification.Attachment{
		{
			Text: fmt.Sprintf("Broadway instance was %s: %s %s.", s, i.PlaybookID, i.ID),
		},
	}
	tp, ok := pb.Messages["created"]
	if update == true {
		tp, ok = pb.Messages["updated"]
	}
	if ok {
		b := new(bytes.Buffer)
		err := template.Must(template.New("created").Parse(tp)).Execute(b, vars(i))
		if err != nil {
			return err
		}
		atts = append(atts, notification.Attachment{
			Text:  b.String(),
			Color: "good",
		})
	}

	m := &notification.Message{
		Attachments: atts,
	}

	return m.Send()
}
