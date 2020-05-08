package paths

type step struct {
	When  conditions             `yaml:"when,omitempty"`
	Attrs map[string]interface{} `yaml:",inline"`
}

func (s *step) matchOrExclude(changedFiles []string) bool {
	if s.When.Paths.HasIncludes() || s.When.Paths.HasExcludes() {
		for _, p := range changedFiles {
			if s.When.Paths.match(p) {
				// we want to run the step
				return false
			}
		}
	}
	// if only When.Paths is set, When.Attrs will be unset, so it must be initialized
	if s.When.Attrs == nil {
		s.When.Attrs = make(map[string]interface{})
	}
	s.When.Attrs["event"] = map[string][]string{"exclude": {"*"}}
	return true
}
