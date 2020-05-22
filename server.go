package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/andrewstucki/drone-infrastructure-plugin/cache"
	"github.com/andrewstucki/drone-infrastructure-plugin/chain"
	"github.com/andrewstucki/drone-infrastructure-plugin/deploy"
	"github.com/andrewstucki/drone-infrastructure-plugin/paths"
	"github.com/aws/aws-sdk-go-v2/aws/ec2metadata"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	admitPlugin "github.com/drone/drone-admit-members/plugin"
	awsSecretPlugin "github.com/drone/drone-amazon-secrets/plugin"
	"github.com/drone/drone-go/plugin/admission"
	"github.com/drone/drone-go/plugin/converter"
	"github.com/drone/drone-go/plugin/secret"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func initializeServer(spec *spec) *http.Server {
	client := setupGithubClient(spec)
	plugin := chain.New().
		WithAdmission(setupAdmission(client, spec)).
		WithConverters(setupConvert(client)).
		WithSecrets(setupSecrets())

	router := http.NewServeMux()
	router.Handle("/admit", plugin.AdmissionHandler(spec.Secret))
	router.Handle("/convert", plugin.ConvertHandler(spec.Secret))
	router.Handle("/secret", plugin.SecretHandler(spec.Secret))
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

func setupSecrets() []secret.Plugin {
	external.LoadDefaultAWSConfig()
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		logrus.Fatalln(err)
	}
	if cfg.Region == "" {
		metaClient := ec2metadata.New(cfg)
		if region, err := metaClient.Region(); err == nil {
			cfg.Region = region
			logrus.Infof("using region %s from ec2 metadata", cfg.Region)
		} else {
			logrus.Fatalf("failed to determine region: %s", err)
		}
	}
	return []secret.Plugin{awsSecretPlugin.New(secretsmanager.New(cfg))}
}

func setupAdmission(client *github.Client, spec *spec) []admission.Plugin {
	// we need to lookup the github team name
	// to gets its unique system identifier
	var team int64
	if spec.Team != "" {
		for {
			result, _, err := client.Teams.GetTeamBySlug(context.Background(), spec.Org, spec.Team)
			if err == context.Canceled {
				break
			}
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					logrus.WithError(err).Warnln("timeout fetching team")
					time.Sleep(time.Second)
				} else {
					logrus.WithError(err).
						WithField("org", spec.Org).
						WithField("team", spec.Team).
						Fatalln("cannot find team")
				}
			} else {
				logrus.Debugln("successfully retrieved team")
				team = result.GetID()
				break
			}
		}
	}

	return []admission.Plugin{admitPlugin.New(client, spec.Org, team)}
}

func setupConvert(client *github.Client) []converter.Plugin {
	return []converter.Plugin{
		cache.New(),
		paths.New(client.Repositories),
		deploy.New(),
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
