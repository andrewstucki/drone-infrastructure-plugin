package main

import (
	"context"
	"sync"
	"time"

	"github.com/andrewstucki/drone-infrastructure-plugin/runner"
	"github.com/drone/signal"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sirupsen/logrus"
)

// spec provides the plugin settings.
type spec struct {
	Bind   string `envconfig:"DRONE_BIND" default:":3000"`
	Debug  bool   `envconfig:"DRONE_DEBUG"`
	Secret string `envconfig:"DRONE_SECRET"`

	// for github auth and path checking
	UseExtensions bool   `envconfig:"DRONE_USE_EXTENSIONS"`
	Token         string `envconfig:"DRONE_GITHUB_TOKEN"`
	Endpoint      string `envconfig:"DRONE_GITHUB_ENDPOINT" default:"https://api.github.com/"`
	Org           string `envconfig:"DRONE_GITHUB_ORG"`
	Team          string `envconfig:"DRONE_GITHUB_TEAM"`

	// gc settings
	UseGC      bool          `envconfig:"DRONE_USE_GC"`
	Images     []string      `envconfig:"DRONE_GC_IGNORE_IMAGES"`
	Containers []string      `envconfig:"DRONE_GC_IGNORE_CONTAINERS"`
	Interval   time.Duration `envconfig:"DRONE_GC_INTERVAL" default:"5m"`
	Cache      string        `envconfig:"DRONE_GC_CACHE" default:"5gb"`

	// runner settings
	UseRunner    bool `envconfig:"DRONE_USE_RUNNER"`
	RunnerConfig runner.Config
}

func main() {
	spec := new(spec)
	err := envconfig.Process("", spec)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.SetLevel(logrus.InfoLevel)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if spec.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	ctx := signal.WithContext(log.Logger.WithContext(context.Background()))

	var wg sync.WaitGroup
	if spec.UseExtensions {
		if spec.Secret == "" {
			logrus.Fatalln("missing secret key")
		}

		wg.Add(1)
		server := initializeServer(spec)
		go func() {
			defer wg.Done()
			runServer(ctx, server, spec)
		}()
	}
	if spec.UseGC {
		collector := initializeGC(ctx, spec)
		wg.Add(1)
		go func() {
			defer wg.Done()
			runGC(ctx, collector, spec)
		}()
	}
	if spec.UseRunner {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runner.Run(ctx, &spec.RunnerConfig); err != nil {
				logrus.WithError(err).Fatalln("error running docker runner")
			}
		}()
	}
	wg.Wait()
}
