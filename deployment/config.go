package deployment

import (
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	"github.com/namely/broadway/env"

	"k8s.io/kubernetes/pkg/client/restclient"
)

// IsKubernetesEnv returns true if necessary Kubernetes environment variables
// and files are available
func IsKubernetesEnv() bool {
	files := []string{
		"/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		"/var/run/secrets/kubernetes.io/serviceaccount/token",
	}

	for _, file := range files {
		fi, err := os.Stat(file)
		if err != nil || fi.IsDir() {
			glog.Infof("Not using kubernetes env because a file is missing: %s", file)
			return false
		}
	}

	envs := []string{
		env.K8sServicePort,
		env.K8sServiceHost,
	}

	for _, env := range envs {
		if env == "" {
			glog.Info("Not running in kub environment because an env var is missing.")
			return false
		}
	}

	return true
}

// Config returns a kubernetes configuration
func Config() (*restclient.Config, error) {
	config := LocalConfig()
	if IsKubernetesEnv() {
		var err error
		config, err = KubernetesConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// KubernetesConfig returns Kubernetes configuration for native Kubernetes environment
func KubernetesConfig() (*restclient.Config, error) {
	token, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, err
	}
	return &restclient.Config{
		Host:        "https://" + env.K8sServiceHost + ":" + env.K8sServicePort,
		BearerToken: string(token),
		TLSClientConfig: restclient.TLSClientConfig{
			CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
	}, nil
}

// LocalConfig returns a configuration for local development
func LocalConfig() *restclient.Config {
	if env.K8sServiceHost != "" && env.K8sCertFile != "" &&
		env.K8sKeyFile != "" && env.K8sCAFile != "" {

		return &restclient.Config{
			Host: env.K8sServiceHost,
			TLSClientConfig: restclient.TLSClientConfig{
				CertFile: env.K8sCertFile,
				KeyFile:  env.K8sKeyFile,
				CAFile:   env.K8sCAFile,
			},
		}
	}

	return &restclient.Config{
		Host:     "http://localhost:8080",
		Insecure: true,
	}
}
