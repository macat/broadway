package notification

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/namely/broadway/env"
)

// Field is a Slack message field
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Attachment is Slack message attachment
type Attachment struct {
	Text       string   `json:"text"`
	Pretext    string   `json:"pretext"`
	Fallback   string   `json:"fallback"`
	Title      string   `json:"title"`
	Fields     []Field  `json:"fields"`
	Color      string   `json:"color"`
	AuthorName string   `json:"author_name"`
	MarkdownIn []string `json:"mrkdwn_in"`
}

// Message is a Slack message
type Message struct {
	ResponseType string       `json:"response_type"`
	Attachments  []Attachment `json:"attachments"`
}

// Send sends the Slack notification
func (message *Message) Send() error {
	if env.SlackWebhook == "" {
		glog.Warning("SLACK_WEBHOOK env var is unset")
		return nil
	}

	value, err := json.Marshal(message)
	if err != nil {
		return err
	}
	glog.Infof("Sending Slack message to %s", env.SlackWebhook)
	resp, err := http.Post(env.SlackWebhook, "application/json", bytes.NewReader(value))
	glog.Info("Slack returned: ", resp)
	return err
}
