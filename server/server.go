package server

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/namely/broadway/deployment"
	"github.com/namely/broadway/env"
	"github.com/namely/broadway/instance"
	"github.com/namely/broadway/manifest"
	"github.com/namely/broadway/playbook"
	"github.com/namely/broadway/services"
	"github.com/namely/broadway/store"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Server provides an HTTP interface to manipulate Playbooks and Instances
type Server struct {
	store      store.Store
	slackToken string
	playbooks  map[string]*playbook.Playbook
	manifests  map[string]*manifest.Manifest
	deployer   deployment.Deployer
	engine     *gin.Engine
}

// ErrorResponse represents a JSON response to be returned in failure cases
type ErrorResponse map[string]string

// BadRequestError represents a JSON response for status 400
var BadRequestError = ErrorResponse{"error": "Bad Request"}

// UnauthorizedError represents a JSON response for status 401
var UnauthorizedError = ErrorResponse{"error": "Unauthorized"}

// NotFoundError represents a JSON response for status 404
var NotFoundError = ErrorResponse{"error": "Not Found"}

// InternalError represents a JSON response for status 500
var InternalError = map[string]string{"error": "Internal Server Error"}

// CustomError creates an ErrorResponse with a custom message
func CustomError(message string) ErrorResponse {
	return ErrorResponse{"error": message}
}

// New instantiates a new Server and binds its handlers. The Server will look
// for playbooks and instances in store `s`
func New(s store.Store) *Server {
	srvr := &Server{
		store:      s,
		slackToken: env.SlackToken,
	}
	srvr.setupHandlers()
	return srvr
}

// Init initializes manifests and playbooks for the server.
func (s *Server) Init() {
	ms := services.NewManifestService("manifests/")

	var err error
	s.manifests, err = ms.LoadManifestFolder()
	if err != nil {
		glog.Fatal(err)
	}

	s.playbooks, err = playbook.LoadPlaybookFolder("playbooks/")
	glog.Info("%+v", s.playbooks)
	if err != nil {
		glog.Fatal(err)
	}
}

func (s *Server) setupHandlers() {
	s.engine = gin.Default()
	gin.SetMode(gin.ReleaseMode) // Comment this to use debug mode for more verbose output
	s.engine.POST("/instances", s.createInstance)
	s.engine.GET("/instance/:playbookID/:instanceID", s.getInstance)
	s.engine.GET("/instances/:playbookID", s.getInstances)
	s.engine.GET("/status/:playbookID/:instanceID", s.getStatus)
	s.engine.POST("/command", s.postCommand)
	s.engine.GET("/command", s.getCommand)
	s.engine.POST("/deploy/:playbookID/:instanceID", s.deployInstance)
}

// Handler returns a reference to the Gin engine that powers Server
func (s *Server) Handler() http.Handler {
	return s.engine
}

// Run starts the server on the specified address
func (s *Server) Run(addr ...string) error {
	return s.engine.Run(addr...)
}

func (s *Server) createInstance(c *gin.Context) {
	var i instance.Instance
	if err := c.BindJSON(&i); err != nil {
		glog.Error(err)
		c.JSON(http.StatusBadRequest, CustomError("Missing: "+err.Error()))
		return
	}

	service := services.NewInstanceService(store.New())
	err := service.Create(&i)

	if err != nil {
		glog.Error(err)
		c.JSON(http.StatusInternalServerError, InternalError)
		return
	}

	c.JSON(http.StatusCreated, i)
}

func (s *Server) getInstance(c *gin.Context) {
	service := services.NewInstanceService(s.store)
	i, err := service.Show(c.Param("playbookID"), c.Param("instanceID"))

	if err != nil {
		switch err.(type) {
		case instance.NotFound:
			c.JSON(http.StatusNotFound, NotFoundError)
			return
		default:
			c.JSON(http.StatusInternalServerError, InternalError)
			return
		}
	}
	c.JSON(http.StatusOK, i)
}

func (s *Server) getInstances(c *gin.Context) {
	service := services.NewInstanceService(s.store)
	instances, err := service.AllWithPlaybookID(c.Param("playbookID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, InternalError)
		return
	}
	c.JSON(http.StatusOK, instances)
	return
}

func (s *Server) getStatus(c *gin.Context) {
	service := services.NewInstanceService(s.store)
	i, err := service.Show(c.Param("playbookID"), c.Param("instanceID"))

	if err != nil {
		switch err.(type) {
		case instance.NotFound:
			c.JSON(http.StatusNotFound, NotFoundError)
			return
		default:
			c.JSON(http.StatusInternalServerError, InternalError)
			return
		}
	}
	c.JSON(http.StatusOK, map[string]string{
		"status": string(i.Status),
	})
}

func (s *Server) getCommand(c *gin.Context) {
	ssl := c.Query("ssl_check")
	glog.Info(ssl)
	if ssl == "1" {
		c.String(http.StatusOK, "")
	} else {
		c.String(http.StatusBadRequest, "Use POST /command")
	}
}

// SlackCommand ...
type SlackCommand struct {
	Token       string `form:"token"`
	TeamID      string `form:"team_id"`
	TeamDomain  string `form:"team_domain"`
	ChannelID   string `form:"channel_id"`
	ChannelName string `form:"channel_name"`
	UserID      string `form:"user_id"`
	UserName    string `form:"user_name"`
	Command     string `form:"command"`
	Text        string `form:"text"`
	ResponseURL string `form:"response_url"`
}

func (s *Server) postCommand(c *gin.Context) {
	var form SlackCommand
	if err := c.BindWith(&form, binding.Form); err != nil {
		glog.Error(err)
		c.JSON(http.StatusInternalServerError, InternalError)
		return
	}

	if form.Token != s.slackToken {
		glog.Errorf("Token mismatch, actual: %s, expected: %s\n", form.Token, s.slackToken)
		c.JSON(http.StatusUnauthorized, UnauthorizedError)
		return
	}
	if form.Text == "help" {
		c.String(http.StatusOK, "/broadway status playbook1 instance1: Check the status of instance1\n /broadway deploy playbook1 instance1: Deploy instance1")
		return
	}
	output, err := helperRunCommand(form.Text)
	if err != nil {
		glog.Error(err)
		c.JSON(http.StatusInternalServerError, InternalError)
		return
	}
	c.String(http.StatusOK, output)
	return
}

func helperRunCommand(text string) (string, error) {
	return "unimplemented :sadpanda:", nil
}

func (s *Server) deployInstance(c *gin.Context) {
	service := services.NewInstanceService(s.store)
	i, err := service.Show(c.Param("playbookID"), c.Param("instanceID"))

	if err != nil {
		glog.Error(err)
		switch err.(type) {
		case instance.NotFound:
			c.JSON(http.StatusNotFound, NotFoundError)
			return
		default:
			c.JSON(http.StatusInternalServerError, InternalError)
			return
		}
	}

	deployService := services.NewDeploymentService(s.store, s.playbooks, s.manifests)

	err = deployService.Deploy(i)
	if err != nil {
		glog.Error(err)
		c.JSON(http.StatusInternalServerError, InternalError)
		return
	}

	c.JSON(http.StatusOK, i)
}
