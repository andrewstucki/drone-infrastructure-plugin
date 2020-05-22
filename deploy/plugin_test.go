package deploy

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/converter"
	"github.com/stretchr/testify/require"
)

var noContext = context.Background()

func TestPlugin(t *testing.T) {
	tests := []struct {
		file string
	}{
		{"pipeline"},
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

			config, err := New().Convert(noContext, req)
			require.NoError(t, err)
			require.NotNil(t, config)
			require.Equal(t, string(after), config.Data)
		})
	}
}
