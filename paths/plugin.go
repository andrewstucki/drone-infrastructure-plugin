package paths

import (
	"bytes"
	"context"
	"io"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/converter"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

//go:generate mockgen -source plugin.go -package paths -destination mock_test.go

// GithubRepositoryClient is an interface for retrieving commits from github
type GithubRepositoryClient interface {
	GetCommit(ctx context.Context, owner string, repo string, sha string) (*github.RepositoryCommit, *github.Response, error)
	CompareCommits(ctx context.Context, owner string, repo string, base string, head string) (*github.CommitsComparison, *github.Response, error)
}

// New returns a new conversion plugin.
func New(client GithubRepositoryClient) converter.Plugin {
	return &plugin{
		client: client,
	}
}

type plugin struct {
	client GithubRepositoryClient
}

func getFilesChanged(ctx context.Context, repo drone.Repo, build drone.Build, client GithubRepositoryClient) ([]string, error) {
	var commitFiles []github.CommitFile
	if build.Before == "" || build.Before == "0000000000000000000000000000000000000000" {
		response, _, err := client.GetCommit(ctx, repo.Namespace, repo.Name, build.After)
		if err != nil {
			return nil, err
		}
		commitFiles = response.Files
	} else {
		response, _, err := client.CompareCommits(ctx, repo.Namespace, repo.Name, build.Before, build.After)
		if err != nil {
			return nil, err
		}
		commitFiles = response.Files
	}

	var files []string
	for _, f := range commitFiles {
		files = append(files, *f.Filename)
	}

	return files, nil
}

func shouldGetFiles(pipelines []*pipeline) bool {
	// we only need to grab the files that changed if we actually have a inclusion/exclusion trigger
	for _, p := range pipelines {
		if p.Kind != "pipeline" {
			continue
		}
		if p.Trigger.Paths.HasIncludes() || p.Trigger.Paths.HasExcludes() {
			return true
		}
		for _, step := range p.Steps {
			if step.When.Paths.HasIncludes() || step.When.Paths.HasExcludes() {
				return true
			}
		}
	}
	return false
}

func (p *plugin) Convert(ctx context.Context, req *converter.Request) (*drone.Config, error) {
	logrus.WithFields(logrus.Fields{
		"build_action":   req.Build.Action,
		"build_after":    req.Build.After,
		"build_before":   req.Build.Before,
		"build_event":    req.Build.Event,
		"build_id":       req.Build.ID,
		"build_number":   req.Build.Number,
		"build_parent":   req.Build.Parent,
		"build_source":   req.Build.Source,
		"build_ref":      req.Build.Ref,
		"build_target":   req.Build.Target,
		"build_trigger":  req.Build.Trigger,
		"repo_namespace": req.Repo.Namespace,
		"repo_name":      req.Repo.Name,
	}).Debugln("initiated path skipping convert plugin")

	pipelines := []*pipeline{}

	decoder := yaml.NewDecoder(bytes.NewBuffer([]byte(req.Config.Data)))
	for {
		pipeline := new(pipeline)
		err := decoder.Decode(pipeline)
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"build_id":       req.Build.ID,
				"repo_namespace": req.Repo.Namespace,
				"repo_name":      req.Repo.Name,
			}).Errorln(err)
			return nil, nil
		}
		pipelines = append(pipelines, pipeline)
	}

	if !shouldGetFiles(pipelines) {
		// no need to go any further since we have no inclusion/exclusion rules
		return &req.Config, nil
	}

	files, err := getFilesChanged(ctx, req.Repo, req.Build, p.client)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"build_id":       req.Build.ID,
			"repo_namespace": req.Repo.Namespace,
			"repo_name":      req.Repo.Name,
		}).Errorln(err)
		return nil, nil
	}

	for _, p := range pipelines {
		if p.update(files) {
			logrus.WithFields(logrus.Fields{
				"build_id":       req.Build.ID,
				"repo_namespace": req.Repo.Namespace,
				"repo_name":      req.Repo.Name,
				"pipeline_name":  p.Name,
			}).Debugln("skipping part of pipeline")
		}
	}

	buffer := new(bytes.Buffer)
	encoder := yaml.NewEncoder(buffer)
	for _, p := range pipelines {
		if err := encoder.Encode(p); err != nil {
			logrus.WithFields(logrus.Fields{
				"build_id":       req.Build.ID,
				"repo_namespace": req.Repo.Namespace,
				"repo_name":      req.Repo.Name,
			}).Errorln(err)
			return nil, nil
		}
	}
	return &drone.Config{
		Data: buffer.String(),
	}, nil
}
