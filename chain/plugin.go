package chain

import (
	"context"
	"net/http"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/admission"
	"github.com/drone/drone-go/plugin/converter"
	"github.com/drone/drone-go/plugin/secret"
	"github.com/sirupsen/logrus"
)

// ChainedPlugin allows you to chain drone plugins
type ChainedPlugin struct {
	converters []converter.Plugin
	admit      []admission.Plugin
	secrets    []secret.Plugin
}

// New initializes a ChainedPlugin instance
func New() *ChainedPlugin {
	return &ChainedPlugin{}
}

// WithConverters adds a series of converter plugins
func (p *ChainedPlugin) WithConverters(converters []converter.Plugin) *ChainedPlugin {
	p.converters = converters
	return p
}

// Convert calls all of the convert plugins that are chained
func (p *ChainedPlugin) Convert(ctx context.Context, req *converter.Request) (*drone.Config, error) {
	for _, c := range p.converters {
		cfg, err := c.Convert(ctx, req)
		if err != nil {
			return nil, err
		}
		if cfg == nil {
			return nil, nil
		}
		req.Config = *cfg
	}
	return &req.Config, nil
}

// ConvertHandler wraps the plugin in a converter handler
func (p *ChainedPlugin) ConvertHandler(secret string) http.Handler {
	return converter.Handler(p, secret, logrus.StandardLogger())
}

// WithAdmission adds a series of admission plugins
func (p *ChainedPlugin) WithAdmission(admit []admission.Plugin) *ChainedPlugin {
	p.admit = admit
	return p
}

// Admit calls all of the admission plugins that are chained
func (p *ChainedPlugin) Admit(ctx context.Context, req *admission.Request) (*drone.User, error) {
	for _, a := range p.admit {
		user, err := a.Admit(ctx, req)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, nil
		}
		req.User = *user
	}
	return &req.User, nil
}

// AdmissionHandler wraps the plugin in an admission handler
func (p *ChainedPlugin) AdmissionHandler(secret string) http.Handler {
	return admission.Handler(p, secret, logrus.StandardLogger())
}

// WithSecrets adds a series of secret plugins
func (p *ChainedPlugin) WithSecrets(secrets []secret.Plugin) *ChainedPlugin {
	p.secrets = secrets
	return p
}

// Find calls all of the secret plugins that are chained and returns the first secret it finds
func (p *ChainedPlugin) Find(ctx context.Context, req *secret.Request) (*drone.Secret, error) {
	for _, s := range p.secrets {
		secret, err := s.Find(ctx, req)
		if err != nil {
			return nil, err
		}
		if secret != nil {
			return secret, nil
		}
	}
	return nil, nil
}

// SecretHandler wraps the plugin in a secret handler
func (p *ChainedPlugin) SecretHandler(s string) http.Handler {
	return secret.Handler(s, p, logrus.StandardLogger())
}
