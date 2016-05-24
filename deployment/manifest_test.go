package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	cases := []struct {
		scenario string
		template string
		vars     map[string]string
		expected string
	}{
		{
			scenario: "simple",
			template: `{{ .test }}`,
			vars:     map[string]string{"test": "hello!"},
			expected: "hello!",
		},
		{
			scenario: "template",
			template: `{{ $cities := split .test ","}}{{ join $cities "+" }}`,
			vars:     map[string]string{"test": "brooklyn,manhattan"},
			expected: "brooklyn+manhattan",
		},
	}

	for _, c := range cases {
		m, err := NewManifest(c.scenario, c.template)
		assert.Nil(t, err)
		out := m.Execute(c.vars)
		assert.Equal(t, c.expected, out, c.scenario+" case does not match output")
	}
}
