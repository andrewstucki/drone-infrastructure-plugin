package runner

import (
	"context"
	"path/filepath"
	"time"

	"github.com/drone-runners/drone-runner-docker/engine"
	"github.com/drone-runners/drone-runner-docker/engine/compiler"
	"github.com/drone-runners/drone-runner-docker/engine/linter"
	"github.com/drone-runners/drone-runner-docker/engine/resource"
	"github.com/drone/drone-go/drone"
	"github.com/drone/runner-go/client"
	"github.com/drone/runner-go/environ/provider"
	"github.com/drone/runner-go/handler/router"
	"github.com/drone/runner-go/logger"
	loghistory "github.com/drone/runner-go/logger/history"
	"github.com/drone/runner-go/pipeline/reporter/history"
	"github.com/drone/runner-go/pipeline/reporter/remote"
	"github.com/drone/runner-go/pipeline/runtime"
	"github.com/drone/runner-go/poller"
	"github.com/drone/runner-go/registry"
	"github.com/drone/runner-go/secret"
	"github.com/drone/runner-go/server"
	"github.com/drone/signal"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func wrapMatch(repos, events []string, trusted bool) func(*drone.Repo, *drone.Build) bool {
	return func(repo *drone.Repo, build *drone.Build) bool {
		// if trusted mode is enabled, only match repositories
		// that are trusted.
		if trusted && repo.Trusted == false {
			return false
		}
		if match(repo.Slug, repos) == false {
			return false
		}
		if match(build.Event, events) == false {
			return false
		}
		return true
	}
}

func match(s string, patterns []string) bool {
	// if no matching patterns are defined the string
	// is always considered a match.
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if match, _ := filepath.Match(pattern, s); match {
			return true
		}
	}
	return false
}

// Run runs the simplified docker runner
func Run(ctx context.Context, config *Config) error {
	config.addDefaults()

	logger.Default = logger.Logrus(
		logrus.NewEntry(
			logrus.StandardLogger(),
		),
	)

	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// listen for termination signals to gracefully shutdown
	// the runner daemon.
	childCtx = signal.WithContextFunc(childCtx, func() {
		println("received signal, terminating process")
		cancel()
	})

	engine, err := engine.NewEnv(engine.Opts{
		HidePull: !config.Docker.Stream,
	})
	if err != nil {
		logrus.WithError(err).Fatalln("cannot load the docker engine")
	}
	for {
		err := engine.Ping(childCtx)
		if err == context.Canceled {
			break
		}
		if err != nil {
			logrus.WithError(err).Errorln("cannot ping the docker daemon")
			time.Sleep(time.Second)
		} else {
			logrus.Debugln("successfully pinged the docker daemon")
			break
		}
	}

	cli := client.New(
		config.Client.Address,
		config.Client.Secret,
		config.Client.SkipVerify,
	)
	cli.Logger = logger.Logrus(
		logrus.NewEntry(
			logrus.StandardLogger(),
		),
	)

	remote := remote.New(cli)
	tracer := history.New(remote)
	hook := loghistory.New()
	logrus.AddHook(hook)

	runner := &runtime.Runner{
		Client:   cli,
		Machine:  config.Runner.Name,
		Reporter: tracer,
		Lookup:   resource.Lookup,
		Lint:     linter.New().Lint,
		Match: wrapMatch(
			config.Limit.Repos,
			config.Limit.Events,
			config.Limit.Trusted,
		),
		Compiler: &compiler.Compiler{
			Clone:      config.Runner.Clone,
			Privileged: append(config.Runner.Privileged, compiler.Privileged...),
			Networks:   config.Runner.Networks,
			Volumes:    config.Runner.Volumes,
			Resources: compiler.Resources{
				Memory:     config.Resources.Memory,
				MemorySwap: config.Resources.MemorySwap,
				CPUQuota:   config.Resources.CPUQuota,
				CPUPeriod:  config.Resources.CPUPeriod,
				CPUShares:  config.Resources.CPUShares,
				CPUSet:     config.Resources.CPUSet,
				ShmSize:    config.Resources.ShmSize,
			},
			Environ:  provider.Static(config.Runner.Environ),
			Registry: registry.File(config.Docker.Config),
			Secret: secret.Combine(
				secret.StaticVars(
					config.Runner.Secrets,
				),
				secret.External(
					"http://localhost:3000/secret",
					config.Client.Secret,
					true,
				),
			),
		},
		Exec: runtime.NewExecer(
			tracer,
			remote,
			engine,
			config.Runner.Procs,
		).Exec,
	}

	poller := &poller.Poller{
		Client:   cli,
		Dispatch: runner.Run,
		Filter: &client.Filter{
			Kind:   resource.Kind,
			Type:   resource.Type,
			OS:     "linux",
			Arch:   "amd64",
			Labels: config.Runner.Labels,
		},
	}

	var group errgroup.Group
	server := server.Server{
		Addr: config.Server.Port,
		Handler: router.New(tracer, hook, router.Config{
			Username: config.Dashboard.Username,
			Password: config.Dashboard.Password,
			Realm:    config.Dashboard.Realm,
		}),
	}

	logrus.WithField("addr", config.Server.Port).Infoln("starting dashboard server")

	group.Go(func() error {
		return server.ListenAndServe(ctx)
	})

	// Ping the server and block until a successful connection
	// to the server has been established.
	for {
		err := cli.Ping(ctx, config.Runner.Name)
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if ctx.Err() != nil {
			break
		}
		if err != nil {
			logrus.WithError(err).
				Errorln("cannot ping the remote server")
			time.Sleep(time.Second)
		} else {
			logrus.Infoln("successfully pinged the remote server")
			break
		}
	}

	group.Go(func() error {
		logrus.WithField("capacity", config.Runner.Capacity).
			WithField("endpoint", config.Client.Address).
			WithField("kind", resource.Kind).
			WithField("type", resource.Type).
			Infoln("polling the remote server")

		poller.Poll(ctx, config.Runner.Capacity)
		return nil
	})

	if err := group.Wait(); err != nil {
		logrus.WithError(err).Errorln("shutting down the server")
		return err
	}
	return nil
}
