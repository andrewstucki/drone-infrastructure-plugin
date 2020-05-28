package deploy

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/converter"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// New returns a new conversion plugin.
func New() converter.Plugin {
	return &plugin{}
}

type plugin struct{}

type pipeline struct {
	Name    string                   `yaml:"name"`
	Kind    string                   `yaml:"kind,omitempty"`
	Steps   []map[string]interface{} `yaml:"steps,omitempty"`
	Volumes []map[string]interface{} `yaml:"volumes,omitempty"`
	Deploy  *struct {
		Repo      string `yaml:"repo"`
		Registry  string `yaml:"registry"`
		Terraform string `yaml:"terraform"`
		Region    string `yaml:"region"`
	} `yaml:"deploy,omitempty"`
	Attrs map[string]interface{} `yaml:",inline"`
}

func (p *pipeline) update() error {
	if p.Deploy != nil {
		terraform := p.Deploy.Terraform
		if terraform == "" {
			terraform = "gracepoint/terraform:0.0.4"
		}
		region := p.Deploy.Region
		if region == "" {
			region = "us-east-1"
		}
		repo := p.Deploy.Repo
		image := fmt.Sprintf("%s/%s:$DRONE_COMMIT", p.Deploy.Registry, p.Deploy.Repo)
		// initialization
		p.Steps = append(p.Steps, map[string]interface{}{
			"name":  "initialize terraform and ecr",
			"image": terraform,
			"commands": []string{
				"decrypt terraform.tfvars.encrypted > terraform.tfvars",
				"terraform init",
				fmt.Sprintf("terraform apply -auto-approve -target aws_ecr_repository.repo -var image=%s", image),
			},
			"environment": map[string]interface{}{
				"AWS_ACCESS_KEY_ID": map[string]interface{}{
					"from_secret": "deploy_access_key",
				},
				"AWS_SECRET_ACCESS_KEY": map[string]interface{}{
					"from_secret": "deploy_secret_key",
				},
			},
		})
		// image publishing
		p.Steps = append(p.Steps, map[string]interface{}{
			"name":  "publish",
			"image": "andrewstucki/plugin-drone-ecr:1",
			"volumes": []map[string]string{
				{
					"name": "docker",
					"path": "/var/run/docker.sock",
				},
			},
			"settings": map[string]interface{}{
				"auto_tag": true,
				"repo":     repo,
				"access_key": map[string]interface{}{
					"from_secret": "deploy_access_key",
				},
				"secret_key": map[string]interface{}{
					"from_secret": "deploy_secret_key",
				},
			},
		})
		// apply
		p.Steps = append(p.Steps, map[string]interface{}{
			"name":  "deploy",
			"image": terraform,
			"commands": []string{
				fmt.Sprintf("terraform apply -auto-approve -var image=%s", image),
				fmt.Sprintf("wait-for-ecs `terraform output cluster` %s", image),
			},
			"environment": map[string]interface{}{
				"AWS_ACCESS_KEY_ID": map[string]interface{}{
					"from_secret": "deploy_access_key",
				},
				"AWS_SECRET_ACCESS_KEY": map[string]interface{}{
					"from_secret": "deploy_secret_key",
				},
				"AWS_REGION": region,
			},
		})

		// add docker volume
		p.Volumes = append(p.Volumes, map[string]interface{}{
			"name": "docker",
			"host": map[string]string{
				"path": "/var/run/docker.sock",
			},
		})

		p.Deploy = nil
	}
	return nil
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

	for _, p := range pipelines {
		if err := p.update(); err != nil {
			logrus.WithFields(logrus.Fields{
				"build_id":       req.Build.ID,
				"repo_namespace": req.Repo.Namespace,
				"repo_name":      req.Repo.Name,
			}).Errorln(err)
			return nil, nil
		}
	}

	pipelines = append(pipelines, &pipeline{
		Name: "deploy_access_key",
		Kind: "secret",
		Attrs: map[string]interface{}{
			"get": map[string]interface{}{
				"path": "drone",
				"name": "deploy-access-key",
			},
		},
	}, &pipeline{
		Name: "deploy_secret_key",
		Kind: "secret",
		Attrs: map[string]interface{}{
			"get": map[string]interface{}{
				"path": "drone",
				"name": "deploy-secret-key",
			},
		},
	})

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
