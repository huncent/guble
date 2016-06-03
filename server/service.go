package server

import (
	"github.com/smancke/guble/protocol"
	"github.com/smancke/guble/server/webserver"

	"fmt"
	"github.com/docker/distribution/health"
	"net/http"
	"reflect"
	"time"
)

const (
	healthEndpointPrefix        = "/health"
	defaultStopGracePeriod      = time.Second * 2
	defaultHealthCheckFrequency = time.Second * 60
	defaultHealthCheckThreshold = 1
)

// Startable interface for modules which provide a start mechanism
type Startable interface {
	Start() error
}

// Stopable interface for modules which provide a stop mechanism
type Stopable interface {
	Stop() error
}

// Endpoint adds a HTTP handler for the `GetPrefix()` to the webserver
type Endpoint interface {
	http.Handler
	GetPrefix() string
}

// Service is the main class for simple control of a server
type Service struct {
	webserver            *webserver.WebServer
	router               Router
	modules              []interface{}
	StopGracePeriod      time.Duration // The timeout given to each Module on Stop()
	healthCheckFrequency time.Duration
	healthCheckThreshold int
}

// NewService registers the Main Router, where other modules can subscribe for messages
func NewService(router Router, webserver *webserver.WebServer) *Service {
	service := &Service{
		webserver:            webserver,
		router:               router,
		StopGracePeriod:      defaultStopGracePeriod,
		healthCheckFrequency: defaultHealthCheckFrequency,
		healthCheckThreshold: defaultHealthCheckThreshold,
	}
	service.registerModule(service.router)
	service.registerModule(service.webserver)

	return service
}

func (s *Service) RegisterModules(modules []interface{}) {
	for _, module := range modules {
		s.registerModule(module)
	}
}

func (s *Service) registerModule(module interface{}) {
	s.modules = append(s.modules, module)
}

// Start checks the modules for the following interfaces and registers and/or starts:
//   Startable:
//   health.Checker:
//   Endpoint: Register the handler function of the Endpoint in the http service at prefix
func (s *Service) Start() error {
	el := protocol.NewErrorList("service: errors occured while starting: ")
	s.webserver.Handle(healthEndpointPrefix, http.HandlerFunc(health.StatusHandler))
	for _, module := range s.modules {
		name := reflect.TypeOf(module).String()
		if startable, ok := module.(Startable); ok {
			protocol.Info("service: starting module %v", name)
			if err := startable.Start(); err != nil {
				protocol.Err("service: error while starting module %v", name)
				el.Add(err)
			}
		}
		if checker, ok := module.(health.Checker); ok {
			protocol.Info("service: registering %v as HealthChecker", name)
			health.RegisterPeriodicThresholdFunc(name, s.healthCheckFrequency, s.healthCheckThreshold, health.CheckFunc(checker.Check))
		}
		if endpoint, ok := module.(Endpoint); ok {
			prefix := endpoint.GetPrefix()
			protocol.Info("service: registering %v as Endpoint to %v", name, prefix)
			s.webserver.Handle(prefix, endpoint)
		}
	}
	return el.ErrorOrNil()
}

func (s *Service) Stop() error {
	stopables := make([]Stopable, 0)
	for _, module := range s.modules {
		name := reflect.TypeOf(module).String()
		if stopable, ok := module.(Stopable); ok {
			protocol.Info("service: %v is Stopable", name)
			stopables = append(stopables, stopable)
		}
	}
	// stopOrder allows the customized stopping of the modules
	// (not necessarily in the reverse order of their Registrations)
	stopOrder := make([]int, len(stopables))
	for i := 0; i < len(stopables); i++ {
		stopOrder[i] = len(stopables) - i - 1
	}
	protocol.Debug("service: stopping %d modules, in order: %v", len(stopOrder), stopOrder)

	errors := make(map[string]error)
	for _, i := range stopOrder {
		name := reflect.TypeOf(stopables[i]).String()
		stoppedC := make(chan bool)
		errorC := make(chan error)
		protocol.Info("service: stopping [%d] %v", i, name)
		go func() {
			err := stopables[i].Stop()
			if err != nil {
				errorC <- err
				return
			}
			stoppedC <- true
		}()
		select {
		case err := <-errorC:
			protocol.Err("service: error while stopping %v: %v", name, err.Error)
			errors[name] = err
		case <-stoppedC:
			protocol.Info("service: stopped %v", name)
		case <-time.After(s.StopGracePeriod):
			errors[name] = fmt.Errorf("service: error while stopping %v: did not stop after timeout %v", name, s.StopGracePeriod)
			protocol.Err(errors[name].Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("service: errors while stopping modules: %q", errors)
	}
	return nil
}

func (s *Service) Modules() []interface{} {
	return s.modules
}

func (s *Service) WebServer() *webserver.WebServer {
	return s.webserver
}

// stop module with a timeout
func stopAsyncTimeout(m Stopable, timeout int) chan error {
	errorC := make(chan error)
	go func() {
	}()
	return errorC
}

// wait for channel to respond or until time expired
func wait() error {
	return nil
}
