package deployment

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/golang/glog"
	"github.com/namely/broadway/env"
)

// ManifestExtension is added to each task manifest item to make a filename
var ManifestExtension = ".yml"

var templateFuncs = template.FuncMap{
	"split":    strings.Split,
	"join":     strings.Join,
	"datetime": time.Now,
	"toUpper":  strings.ToUpper,
	"toLower":  strings.ToLower,
	"contains": strings.Contains,
	"replace":  strings.Replace,
}

// Manifest represents a kubernetes manifest file
// Filename is used as identifier in the current implementation.
type Manifest struct {
	ID       string
	Template *template.Template
}

// NewManifest creates a new Manifest object and parses (but does not execute) the template
func NewManifest(id, content string) (*Manifest, error) {
	t, err := template.New(id).Funcs(templateFuncs).Parse(content)
	if err != nil {
		return nil, err
	}

	return &Manifest{ID: id, Template: t}, nil
}

// Execute executes template with variables
func (m *Manifest) Execute(vars map[string]string) string {
	var b bytes.Buffer
	err := m.Template.Execute(&b, vars)
	if err != nil {
		glog.Errorf("%s template errored, ignoring: %s", m.ID, err.Error())
		return ""
	}
	return b.String()
}

// ManifestsPresent iterates through the Manifests and PodManifest items on a
// task, and checks that each represents a file on disk
func (t Task) ManifestsPresent() error {
	for _, name := range t.Manifests {
		filename := name + ManifestExtension
		path := filepath.Join(env.ManifestsPath, filename)
		if _, err := os.Stat(path); err != nil {
			return err
		}
	}
	if len(t.PodManifest) > 0 {
		filename := t.PodManifest + ManifestExtension
		path := filepath.Join(env.ManifestsPath, filename)
		if _, err := os.Stat(path); err != nil {
			return err
		}
	}
	return nil
}
