package deployment

// Deployer declares something that can Deploy Deployments
type Deployer interface {
	Deploy() error
	Destroy() error
}
