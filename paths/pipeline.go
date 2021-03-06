package paths

type pipeline struct {
	Name    string                 `yaml:"name"`
	Kind    string                 `yaml:"kind,omitempty"`
	Steps   []*step                `yaml:"steps,omitempty"`
	Trigger conditions             `yaml:"trigger,omitempty"`
	Attrs   map[string]interface{} `yaml:",inline"`
}

func (p *pipeline) matchOrExclude(changedFiles []string) bool {
	for _, f := range changedFiles {
		if p.Trigger.Paths.match(f) {
			return false
		}
	}
	if p.Trigger.Attrs == nil {
		p.Trigger.Attrs = make(map[string]interface{})
	}
	p.Trigger.Attrs["event"] = map[string][]string{"exclude": {"*"}}
	return true
}

func (p *pipeline) update(changedFiles []string) bool {
	if p.Kind != "pipeline" {
		return false
	}

	updated := false
	if p.matchOrExclude(changedFiles) {
		updated = true
	}
	for _, s := range p.Steps {
		if s.matchOrExclude(changedFiles) {
			updated = true
		}
	}
	return updated
}
