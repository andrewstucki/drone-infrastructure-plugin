package main

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/andrewstucki/drone-infrastructure-plugin/cache"
	"github.com/andrewstucki/drone-infrastructure-plugin/chain"
	"github.com/andrewstucki/drone-infrastructure-plugin/paths"
	admitPlugin "github.com/drone/drone-admit-members/plugin"
	"github.com/drone/drone-go/plugin/admission"
	"github.com/drone/drone-go/plugin/converter"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func initializeServer(spec *spec) *http.Server {
	client := setupGithubClient(spec)
	plugin := chain.New().
		WithAdmission(setupAdmission(client, spec)).
		WithConverters(setupConvert(client))

	router := http.NewServeMux()
	router.Handle("/admit", plugin.AdmissionHandler(spec.Secret))
	router.Handle("/convert", plugin.ConvertHandler(spec.Secret))
	router.HandleFunc("/healthz", healthz)

	return &http.Server{
		Addr:    spec.Bind,
		Handler: router,
	}
}

func setupGithubClient(spec *spec) *github.Client {
	// creates the github client transport used
	// to authenticate API requests.
	trans := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: spec.Token},
	))

	// create the github client
	client, err := github.NewEnterpriseClient(spec.Endpoint, spec.Endpoint, trans)
	if err != nil {
		logrus.Fatal(err)
	}

	return client
}

func setupAdmission(client *github.Client, spec *spec) []admission.Plugin {
	// we need to lookup the github team name
	// to gets its unique system identifier
	var team int64
	if spec.Team != "" {
		result, _, err := client.Teams.GetTeamBySlug(context.Background(), spec.Org, spec.Team)
		if err != nil {
			logrus.WithError(err).
				WithField("org", spec.Org).
				WithField("team", spec.Team).
				Fatalln("cannot find team")
		}
		team = result.GetID()
	}

	return []admission.Plugin{admitPlugin.New(client, spec.Org, team)}
}

func setupConvert(client *github.Client) []converter.Plugin {
	return []converter.Plugin{
		cache.New(),
		paths.New(client.Repositories),
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "OK")
}

func runServer(ctx context.Context, server *http.Server, spec *spec) {
	logrus.WithFields(logrus.Fields{
		"bind": spec.Bind,
	}).Infoln("starting plugin server")

	go func() {
		// wait until the context says to stop everything
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logrus.WithError(err).Warnln("server shutdown error")
		}
	}()
	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			logrus.Infoln("server shutting down")
		} else {
			logrus.WithError(err).Fatalln("server prematurely stopped")
		}
	}
}
