package runner

import (
	"fmt"
	"os"
)

// Config is the runner configuration
type Config struct {
	Client struct {
		Address    string `ignored:"true"`
		Proto      string `envconfig:"DRONE_RPC_PROTO"  default:"http"`
		Host       string `envconfig:"DRONE_RPC_HOST"   required:"true"`
		Secret     string `envconfig:"DRONE_RPC_SECRET" required:"true"`
		SkipVerify bool   `envconfig:"DRONE_RPC_SKIP_VERIFY"`
		Dump       bool   `envconfig:"DRONE_RPC_DUMP_HTTP"`
		DumpBody   bool   `envconfig:"DRONE_RPC_DUMP_HTTP_BODY"`
	}

	Dashboard struct {
		Disabled bool   `envconfig:"DRONE_UI_DISABLE"`
		Username string `envconfig:"DRONE_UI_USERNAME"`
		Password string `envconfig:"DRONE_UI_PASSWORD"`
		Realm    string `envconfig:"DRONE_UI_REALM" default:"MyRealm"`
	}

	Server struct {
		Port  string `envconfig:"DRONE_HTTP_BIND" default:":4000"`
		Proto string `envconfig:"DRONE_HTTP_PROTO"`
		Host  string `envconfig:"DRONE_HTTP_HOST"`
		Acme  bool   `envconfig:"DRONE_HTTP_ACME"`
	}

	Runner struct {
		Name       string            `envconfig:"DRONE_RUNNER_NAME"`
		Capacity   int               `envconfig:"DRONE_RUNNER_CAPACITY" default:"2"`
		Procs      int64             `envconfig:"DRONE_RUNNER_MAX_PROCS"`
		Environ    map[string]string `envconfig:"DRONE_RUNNER_ENVIRON"`
		EnvFile    string            `envconfig:"DRONE_RUNNER_ENV_FILE"`
		Secrets    map[string]string `envconfig:"DRONE_RUNNER_SECRETS"`
		Labels     map[string]string `envconfig:"DRONE_RUNNER_LABELS"`
		Volumes    map[string]string `envconfig:"DRONE_RUNNER_VOLUMES"`
		Devices    []string          `envconfig:"DRONE_RUNNER_DEVICES"`
		Networks   []string          `envconfig:"DRONE_RUNNER_NETWORKS"`
		Privileged []string          `envconfig:"DRONE_RUNNER_PRIVILEGED_IMAGES"`
		Clone      string            `envconfig:"DRONE_RUNNER_CLONE_IMAGE"`
	}

	Limit struct {
		Repos   []string `envconfig:"DRONE_LIMIT_REPOS"`
		Events  []string `envconfig:"DRONE_LIMIT_EVENTS"`
		Trusted bool     `envconfig:"DRONE_LIMIT_TRUSTED"`
	}

	Resources struct {
		Memory     int64    `envconfig:"DRONE_MEMORY_LIMIT"`
		MemorySwap int64    `envconfig:"DRONE_MEMORY_SWAP_LIMIT"`
		CPUQuota   int64    `envconfig:"DRONE_CPU_QUOTA"`
		CPUPeriod  int64    `envconfig:"DRONE_CPU_PERIOD"`
		CPUShares  int64    `envconfig:"DRONE_CPU_SHARES"`
		CPUSet     []string `envconfig:"DRONE_CPU_SET"`
		ShmSize    int64    `envconfig:"DRONE_SHM_SIZE"`
	}

	Docker struct {
		Config string `envconfig:"DRONE_DOCKER_CONFIG"`
		Stream bool   `envconfig:"DRONE_DOCKER_STREAM_PULL" default:"true"`
	}

	Secret struct {
		Endpoint string `envconfig:"DRONE_SECRET_PLUGIN_ENDPOINT"`
		Token    string `envconfig:"DRONE_SECRET_PLUGIN_TOKEN"`
	}
}

func (c *Config) addDefaults() {
	if c.Runner.Environ == nil {
		c.Runner.Environ = map[string]string{}
	}
	if c.Runner.Name == "" {
		c.Runner.Name, _ = os.Hostname()
	}
	if c.Dashboard.Password == "" {
		c.Dashboard.Disabled = true
	}
	c.Client.Address = fmt.Sprintf(
		"%s://%s",
		c.Client.Proto,
		c.Client.Host,
	)
}
