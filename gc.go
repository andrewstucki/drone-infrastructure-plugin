package main

import (
	"context"

	"docker.io/go-docker"
	"github.com/docker/go-units"
	"github.com/drone/drone-gc/gc"
	"github.com/drone/drone-gc/gc/cache"
	"github.com/sirupsen/logrus"
)

func initializeGC(ctx context.Context, spec *spec) gc.Collector {
	client, err := docker.NewEnvClient()
	if err != nil {
		logrus.WithError(err).
			Fatalln("cannot create Docker client")
	}

	size, err := units.FromHumanSize(spec.Cache)
	if err != nil {
		logrus.WithError(err).
			Fatalln("cannot parse cache size")
	}

	return gc.New(
		cache.Wrap(ctx, client),
		gc.WithImageWhitelist(gc.ReservedImages),
		gc.WithImageWhitelist(spec.Images),
		gc.WithThreshold(size),
		gc.WithWhitelist(gc.ReservedNames),
		gc.WithWhitelist(spec.Containers),
	)
}

func runGC(ctx context.Context, collector gc.Collector, spec *spec) {
	logrus.WithFields(logrus.Fields{
		"ignore-containers": spec.Containers,
		"ignore-images":     spec.Images,
		"cache":             spec.Cache,
		"interval":          units.HumanDuration(spec.Interval),
	}).Infoln("starting the garbage collector")

	gc.Schedule(ctx, collector, spec.Interval)
}
