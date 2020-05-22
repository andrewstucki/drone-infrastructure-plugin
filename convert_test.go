package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/andrewstucki/drone-infrastructure-plugin/cache"
	"github.com/andrewstucki/drone-infrastructure-plugin/chain"
	"github.com/andrewstucki/drone-infrastructure-plugin/deploy"
	"github.com/andrewstucki/drone-infrastructure-plugin/paths"
	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/converter"
	gomock "github.com/golang/mock/gomock"
	github "github.com/google/go-github/v28/github"
	"github.com/stretchr/testify/require"
)

var noContext = context.Background()

type getCommitResponse struct {
	commit *github.RepositoryCommit
	err    error
}

type compareCommitsResponse struct {
	diff *github.CommitsComparison
	err  error
}

func makeCommitFiles(paths []string) []github.CommitFile {
	files := []github.CommitFile{}
	for _, p := range paths {
		files = append(files, github.CommitFile{
			Filename: &p,
		})
	}
	return files
}

func newGetCommitResponse(paths []string, err error) *getCommitResponse {
	return &getCommitResponse{
		commit: &github.RepositoryCommit{
			Files: makeCommitFiles(paths),
		},
		err: err,
	}
}

func newCompareCommitsResponse(paths []string, err error) *compareCommitsResponse {
	return &compareCommitsResponse{
		diff: &github.CommitsComparison{
			Files: makeCommitFiles(paths),
		},
		err: err,
	}
}

func TestPlugin(t *testing.T) {
	tests := []struct {
		file           string
		commitResponse *getCommitResponse
		diffResponse   *compareCommitsResponse
	}{
		{"pipeline", nil, newCompareCommitsResponse([]string{"README.md"}, nil)},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			beforeFile := fmt.Sprintf("testdata/%s.yml", test.file)
			afterFile := fmt.Sprintf("testdata/%s.yml.golden", test.file)
			before, err := ioutil.ReadFile(beforeFile)
			require.NoError(t, err)
			after, err := ioutil.ReadFile(afterFile)
			require.NoError(t, err)

			build := drone.Build{
				After: "3d21ec53a331a6f037a91c368710b99387d012c1",
			}
			if test.diffResponse != nil {
				build.Before = "1"
			}
			repo := drone.Repo{
				Slug:   "octocat/hello-world",
				Config: ".drone.yml",
			}
			req := &converter.Request{
				Build: build,
				Repo:  repo,
				Config: drone.Config{
					Data: string(before),
				},
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := NewMockGithubRepositoryClient(ctrl)
			if test.commitResponse != nil {
				client.EXPECT().
					GetCommit(noContext, repo.Namespace, repo.Name, build.After).
					Return(test.commitResponse.commit, nil, test.commitResponse.err)
			}
			if test.diffResponse != nil {
				client.EXPECT().
					CompareCommits(noContext, repo.Namespace, repo.Name, build.Before, build.After).
					Return(test.diffResponse.diff, nil, test.diffResponse.err)
			}
			plugin := chain.New().WithConverters([]converter.Plugin{
				cache.New(),
				paths.New(client),
				deploy.New(),
			})
			config, err := plugin.Convert(noContext, req)
			require.NoError(t, err)
			require.NotNil(t, config)
			require.Equal(t, string(after), config.Data)
		})
	}
}
