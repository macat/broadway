package env

import (
	"os"

	"github.com/golang/glog"
)

var (
	// SlackToken contains the expected Slack custom command token.
	SlackToken string

	// ServerHost is passed to gin and configures the listen address of the server
	ServerHost string

	// KubernetesPort represents KUBERNETES_PORT variable
	KubernetesPort string

	// K8sNamespace is the namespace used by Broadway's deployments
	K8sNamespace string

	// EtcdHost is the Etcd host
	EtcdHost string

	// EtcdPath is the root directory for Broadway objects
	EtcdPath string

	// SlackWebhook is the Slack incoming webhook
	SlackWebhook string
)

// LoadEnvs sets env.* variables to their OS-provided value
func LoadEnvs() {
	SlackWebhook = loadw("SLACK_WEBHOOK")
	SlackToken = loadw("SLACK_VERIFICATION_TOKEN")
	ServerHost = loadw("HOST")
	KubernetesPort = loadw("KUBERNETES_PORT")
	K8sNamespace = loadf("KUBERNETES_NAMESPACE")
	EtcdHost = loadw("ETCD_HOST")
	EtcdPath = loadw("ETCD_PATH")
}

func loadw(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		glog.Warningf("Environment variable %s unset; defaulting to empty string\n", key)
	}
	return val
}

func loadf(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		glog.Fatalf("Environment variable %s unset; Exiting.n", key)
	}
	return val
}

func init() {
	LoadEnvs()
}
