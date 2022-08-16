package service

import (
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"os"
)

// NewFlowServiceBuilder implements a new flow service that can have two items.
// Modules build dependencies.
// Components run features and wait until they finish. This is usually done at SIGINT.
func NewFlowServiceBuilder(name string) *FlowServiceBuilder {
	return &FlowServiceBuilder{
		ServiceConfig: ServiceConfig{
			Name:   name,
			Logger: zerolog.New(os.Stderr),
		},
	}
}

type FlowService interface {
	Run()
}

type BuilderFunc func(serviceConfig *ServiceConfig) error

type namedModuleFunc struct {
	fn   BuilderFunc
	name string
}

type ServiceConfig struct {
	Name       string
	Logger     zerolog.Logger
	Flags      pflag.FlagSet
	Components []namedModuleFunc
}

type FlowServiceBuilder struct {
	FlowService
	ServiceConfig ServiceConfig
	modules       []namedModuleFunc // Modules are dependencies built at startup
}

// ParseAndPrintFlags parses and prints command line configuration parameters.
func (fsb *FlowServiceBuilder) ParseAndPrintFlags() error {
	flags := &fsb.ServiceConfig.Flags

	// parse all flags
	err := flags.Parse(os.Args[1:])
	if err != nil {
		fsb.ServiceConfig.Logger.Fatal().Err(err)
	}

	// print all flags
	flags.VisitAll(func(flag *pflag.Flag) {
		fsb.ServiceConfig.Logger.Info().Str(flag.Name, flag.Value.String()).Msg("flags loaded")
	})

	return nil
}

// Build runs all module callbacks.
// This is done once at startup.
func (fsb *FlowServiceBuilder) Build() (*FlowService, error) {
	// build all modules
	for _, f := range fsb.modules {
		if err := f.fn(&fsb.ServiceConfig); err != nil {
			fsb.ServiceConfig.Logger.Err(err)
			return nil, err
		}
		fsb.ServiceConfig.Logger.Info().Str("module", f.name).Msg("service module started")
	}
	return &fsb.FlowService, nil
}

// Module enables setting up dependencies of the engine with the builder context.
// The function is called once when the service starts.
// It is supposed to do its job and finish before components start.
func (fsb *FlowServiceBuilder) Module(name string, f BuilderFunc) *FlowServiceBuilder {
	fsb.modules = append(fsb.modules, namedModuleFunc{
		fn:   f,
		name: name,
	})
	return fsb
}

// Component adds a new component to the node that conforms to the ReadyDoneAware
// interface, and throws a Fatal() when an irrecoverable error is encountered.
func (fsb *FlowServiceBuilder) Component(name string, f BuilderFunc) *FlowServiceBuilder {
	fsb.ServiceConfig.Components = append(fsb.ServiceConfig.Components, namedModuleFunc{
		fn:   f,
		name: name,
	})
	return fsb
}

// Start starts each component and errors out if one fails
func (fsc *ServiceConfig) Start() error {
	// start all components
	for _, f := range fsc.Components {
		if err := f.fn(fsc); err != nil {
			fsc.Logger.Error().Str("component", f.name).Err(err)
			return err
		}
		fsc.Logger.Info().Str("component", f.name).Msg("Service Component Started")
	}

	return nil
}
