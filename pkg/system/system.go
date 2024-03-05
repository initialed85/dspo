package system

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/initialed85/dspo/internal"
	"github.com/initialed85/dspo/pkg/common"
	_fanin "github.com/initialed85/dspo/pkg/fanin"
	_fanout "github.com/initialed85/dspo/pkg/fanout"
	"github.com/initialed85/dspo/pkg/managed_process"
	"github.com/initialed85/dspo/pkg/service"
)

const (
	depth = 65536
)

type System struct {
	serviceArgs   []common.ServiceArgs
	name          string
	mu            *sync.Mutex
	started       bool
	serviceByName map[string]*service.Service
	logger        *slog.Logger
	consumer      chan managed_process.Log
	fanin         *_fanin.Fanin
	fanout        *_fanout.Fanout
}

func New(
	serviceArgs []common.ServiceArgs,
	name string,
) *System {
	s := System{
		serviceArgs:   serviceArgs,
		mu:            new(sync.Mutex),
		started:       false,
		serviceByName: make(map[string]*service.Service),
		logger:        internal.GetLogger(name),
		consumer:      make(chan managed_process.Log, depth),
	}

	s.fanin = _fanin.New(s.consumer)
	s.fanout = _fanout.New(s.consumer)

	return &s
}

func (s *System) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("cannot start, already running")
	}

	//
	// sanity check for duplicates or unknown dependencies
	//

	serviceArgsByName := make(map[string]common.ServiceArgs)

	for _, serviceArgs := range s.serviceArgs {
		_, ok := serviceArgsByName[serviceArgs.Name]
		if ok {
			return fmt.Errorf("duplicate service name %#+v", serviceArgs.Name)
		}

		serviceArgsByName[serviceArgs.Name] = serviceArgs
	}

	for _, serviceArgs := range serviceArgsByName {
		if serviceArgs.DependsOn == nil {
			continue
		}

		for _, name := range serviceArgs.DependsOn {
			_, ok := serviceArgsByName[name]
			if !ok {
				return fmt.Errorf("service %#+v depends on unknown service %#+v", serviceArgs.Name, name)
			}
		}
	}

	//
	// wire everything together in the correct order sanity checking the graph-ish piece
	//

	unhandledServiceArgsByName := make(map[string]common.ServiceArgs)

	for name, serviceArgs := range serviceArgsByName {
		unhandledServiceArgsByName[name] = serviceArgs
	}

	rootServiceByName := make(map[string]*service.Service)
	nonRootServicesByDependsOnName := make(map[string][]*service.Service)
	handledServiceByName := make(map[string]*service.Service)

	// shouldn't be more layers than there are services (would a real graph be better? yes, yes it would)
	for i := 0; i < len(serviceArgsByName); i++ {
		for name, serviceArgs := range unhandledServiceArgsByName {
			name := name
			serviceArgs := serviceArgs

			_, ok := handledServiceByName[name]
			if ok {
				continue
			}

			foundAllDependencies := true

			for _, dependsOn := range serviceArgs.DependsOn {
				_, ok = handledServiceByName[dependsOn]
				if !ok {
					foundAllDependencies = false
					break
				}
			}

			if !foundAllDependencies {
				continue
			}

			actualService := service.New(
				serviceArgs.ManagedProcessArgs,
				serviceArgs.StartupProbeArgs,
				serviceArgs.LivenessProbeArgs,
				func() {
					nonRootServices, ok := nonRootServicesByDependsOnName[serviceArgs.Name]
					if !ok {
						return
					}

					for _, nonRootService := range nonRootServices {
						s.logger.Debug(
							fmt.Sprintf("%v starting non-root service %v", name, nonRootService.Name()),
						)

						_ = nonRootService.Start()

						consumer, unsubscribe, err := nonRootService.SubscribeToLogs()
						if err != nil {
							s.logger.Error(
								"unexpectedly failed to subscribe to logs for service",
								"service", nonRootService.Name(),
								"error", err,
							)
							continue
						}

						s.fanin.Consume(consumer, unsubscribe)
					}
				},
				common.NoOpFunc,
				common.NoOpFunc,
				serviceArgs.Name,
			)

			if len(serviceArgs.DependsOn) == 0 {
				rootServiceByName[name] = actualService
			} else {
				for _, dependsOnName := range serviceArgs.DependsOn {
					nonRootServices, ok := nonRootServicesByDependsOnName[dependsOnName]
					if !ok {
						nonRootServices = make([]*service.Service, 0)
					}

					nonRootServicesByDependsOnName[dependsOnName] = append(nonRootServices, actualService)
				}
			}

			handledServiceByName[name] = actualService

			delete(unhandledServiceArgsByName, name)
		}
	}

	if len(unhandledServiceArgsByName) > 0 {
		return fmt.Errorf(
			"probable graph cycle; couldn't resolve dependencies for %v/%v services after %v iterations",
			len(unhandledServiceArgsByName),
			len(serviceArgsByName),
			len(serviceArgsByName),
		)
	}

	//
	// start the root services (which will start the non-root services)
	//

	for _, actualService := range rootServiceByName {
		s.logger.Debug(fmt.Sprintf("starting root service %v", actualService.Name()))

		_ = actualService.Start()

		consumer, unsubscribe, err := actualService.SubscribeToLogs()
		if err != nil {
			s.logger.Error(
				"unexpectedly failed to subscribe to logs for service",
				"service", actualService.Name(),
				"error", err,
			)
			continue
		}

		s.fanin.Consume(consumer, unsubscribe)
	}

	s.serviceByName = handledServiceByName

	s.started = true

	return nil
}

func (s *System) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("cannot stop, not running")
	}

	s.started = false

	for _, actualService := range s.serviceByName {
		_ = actualService.Stop()
	}

	s.serviceByName = make(map[string]*service.Service)

	s.fanout.Close()
	s.fanin.Close()

	return nil
}

func (s *System) ServiceByName() map[string]*service.Service {
	s.mu.Lock()
	defer s.mu.Unlock()

	serviceByName := make(map[string]*service.Service)

	for _, actualService := range s.serviceByName {
		serviceByName[actualService.Name()] = actualService
	}

	return serviceByName
}

func (s *System) SubscribeToLogs() (chan managed_process.Log, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fanout == nil {
		return nil, nil, fmt.Errorf("no fanout to subscribe to (not started?)")
	}

	logs, cancel := s.fanout.Subscribe()

	return logs, cancel, nil
}
