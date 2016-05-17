package services

import (
	"bytes"
	"fmt"
	"regexp"
	"text/template"

	"github.com/golang/glog"
	"github.com/namely/broadway/deployment"
	"github.com/namely/broadway/instance"
	"github.com/namely/broadway/notification"
	"github.com/namely/broadway/store"
)

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

// Create a new instance
func (is *InstanceService) Create(i *instance.Instance) (*instance.Instance, error) {
	sanitizer, err := regexp.Compile(`[^a-zA-Z0-9\-]`)
	if err != nil {
		panic(err)
	}
	validator, err := regexp.Compile(`^[a-zA-Z0-9\-]{1,253}$`)
	if err != nil {
		panic(err)
	}
	match := validator.FindStringIndex(i.ID)
	if match == nil {
		x := sanitizer.ReplaceAllString(i.ID, "-")
		if len(x) > 253 {
			x = x[0:253]
		}
		return nil, &InvalidID{
			badID:       i.ID,
			suggestedID: x,
		}
	}

	pb, ok := deployment.AllPlaybooks[i.PlaybookID]
	if !ok {
		return nil, &PlaybookNotFound{i.PlaybookID}
	}
	// Set all vars declared in playbook to default empty string
	vars := make(map[string]string)
	for _, pv := range pb.Vars {
		vars[pv] = ""
	}
	// Abort if new instance tries to set vars not declared in playbook
	for k, v := range i.Vars {
		_, valid := vars[k]
		if !valid { // k is not listed in playbook
			return nil, &InvalidVar{i.PlaybookID, k}
		}
		vars[k] = v
	}

	i.Vars = vars
	err = is.repo.Save(i)
	if err != nil {
		return nil, err
	}
	err = sendCreationNotification(i)
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

func sendCreationNotification(i *instance.Instance) error {
	pb, ok := deployment.AllPlaybooks[i.PlaybookID]
	if !ok {
		return fmt.Errorf("Failed to lookup playbook for instance %+v", *i)
	}

	atts := []notification.Attachment{
		{
			Text: fmt.Sprintf("New broadway instance was created: %s %s.", i.PlaybookID, i.ID),
		},
	}
	tp, ok := pb.Messages["created"]
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
