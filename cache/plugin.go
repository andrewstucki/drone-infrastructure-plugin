package cache

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/converter"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type cache struct {
	Hash string `yaml:"hash"` // the path to use for constructing a hash key
	Path string `yaml:"path"` // the path of the location to cache
	TTL  int    `yaml:"ttl"`  // the time the cache will be kept around
}

type stage struct {
	Name  string                   `yaml:"name"`
	Kind  string                   `yaml:"kind,omitempty"`
	Steps []map[string]interface{} `yaml:"steps,omitempty"`
	Cache []cache                  `yaml:"cache,omitempty"`
	Attrs map[string]interface{}   `yaml:",inline"`
}

func (s *stage) update() bool {
	if s.Kind != "pipeline" || len(s.Cache) == 0 {
		return false
	}
	restoreSteps := []map[string]interface{}{}
	storeSteps := []map[string]interface{}{}
	for _, c := range s.Cache {
		if c.Path == "" {
			// skip things where we don't have the two required entry
			continue
		}
		ttl := c.TTL
		if ttl <= 0 {
			ttl = 5 // days
		}
		restoreSteps = append(restoreSteps, map[string]interface{}{
			"name":  fmt.Sprintf("Restore %s", c.Path),
			"image": "andrewstucki/s3-cache",
			"settings": map[string]interface{}{
				"pull":    true,
				"restore": true,
				"hash":    c.Hash,
				"root": map[string]interface{}{
					"from_secret": "cache_bucket",
				},
				"access_key": map[string]interface{}{
					"from_secret": "cache_access_key",
				},
				"secret_key": map[string]interface{}{
					"from_secret": "cache_secret_key",
				},
			},
		})
		// the rebuild step
		storeSteps = append(storeSteps, map[string]interface{}{
			"name":  fmt.Sprintf("Uploading %s", c.Path),
			"image": "andrewstucki/s3-cache",
			"settings": map[string]interface{}{
				"pull":    true,
				"rebuild": true,
				"hash":    c.Hash,
				"mount":   []string{c.Path},
				"root": map[string]interface{}{
					"from_secret": "cache_bucket",
				},
				"access_key": map[string]interface{}{
					"from_secret": "cache_access_key",
				},
				"secret_key": map[string]interface{}{
					"from_secret": "cache_secret_key",
				},
			},
		})
		// the flush step
		storeSteps = append(storeSteps, map[string]interface{}{
			"name":  fmt.Sprintf("Setting TTL for %s", c.Path),
			"image": "andrewstucki/s3-cache",
			"settings": map[string]interface{}{
				"pull":      true,
				"flush":     true,
				"hash":      c.Hash,
				"flush_age": ttl,
				"root": map[string]interface{}{
					"from_secret": "cache_bucket",
				},
				"access_key": map[string]interface{}{
					"from_secret": "cache_access_key",
				},
				"secret_key": map[string]interface{}{
					"from_secret": "cache_secret_key",
				},
			},
		})
	}

	// append the restore steps onto the beginning of the steps
	s.Steps = append(restoreSteps, s.Steps...)
	// append the store steps onto the end of the steps
	s.Steps = append(s.Steps, storeSteps...)

	// clear out the cache
	s.Cache = []cache{}

	return true
}

// New returns a new conversion plugin.
func New() converter.Plugin {
	return &plugin{}
}

type plugin struct{}

// Convert adds caching steps to the configuration
func (p *plugin) Convert(ctx context.Context, req *converter.Request) (*drone.Config, error) {
	stages := []*stage{}

	decoder := yaml.NewDecoder(bytes.NewBuffer([]byte(req.Config.Data)))
	for {
		stage := new(stage)
		err := decoder.Decode(stage)
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
		stages = append(stages, stage)
	}

	for _, s := range stages {
		if s.update() {
			logrus.WithFields(logrus.Fields{
				"build_id":       req.Build.ID,
				"repo_namespace": req.Repo.Namespace,
				"repo_name":      req.Repo.Name,
				"stage_name":     s.Name,
			}).Debugln("updated cache settings for stage")
		}
	}

	stages = append(stages, &stage{
		Name: "cache_access_key",
		Kind: "secret",
		Attrs: map[string]interface{}{
			"get": map[string]interface{}{
				"path": "drone",
				"name": "cache-access-key",
			},
		},
	}, &stage{
		Name: "cache_secret_key",
		Kind: "secret",
		Attrs: map[string]interface{}{
			"get": map[string]interface{}{
				"path": "drone",
				"name": "cache-secret-key",
			},
		},
	}, &stage{
		Name: "cache_bucket",
		Kind: "secret",
		Attrs: map[string]interface{}{
			"get": map[string]interface{}{
				"path": "drone",
				"name": "cache-bucket",
			},
		},
	})
	buffer := new(bytes.Buffer)
	encoder := yaml.NewEncoder(buffer)
	for _, s := range stages {
		if err := encoder.Encode(s); err != nil {
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
