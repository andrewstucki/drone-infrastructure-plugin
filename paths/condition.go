package paths

import (
	filepath "github.com/bmatcuk/doublestar"
)

type condition struct {
	Exclude []string `yaml:"exclude,omitempty"`
	Include []string `yaml:"include,omitempty"`
}

func (c *condition) HasIncludes() bool {
	return len(c.Include) > 0
}

func (c *condition) HasExcludes() bool {
	return len(c.Exclude) > 0
}

// match returns true if the string matches the include
// patterns and does not match any of the exclude patterns.
func (c *condition) match(v string) bool {
	if c.excludes(v) {
		return false
	}
	if c.includes(v) {
		return true
	}
	if len(c.Include) == 0 {
		return true
	}
	return false
}

// includes returns true if the string matches the include
// patterns.
func (c *condition) includes(v string) bool {
	for _, pattern := range c.Include {
		if ok, _ := filepath.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

// excludes returns true if the string matches the exclude
// patterns.
func (c *condition) excludes(v string) bool {
	for _, pattern := range c.Exclude {
		if ok, _ := filepath.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

type conditions struct {
	Paths condition              `yaml:"paths,omitempty"`
	Attrs map[string]interface{} `yaml:",inline"`
}
