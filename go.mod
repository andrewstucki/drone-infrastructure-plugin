module github.com/andrewstucki/drone-infrastructure-plugin

go 1.12

require (
	docker.io/go-docker v1.0.0
	github.com/aws/aws-sdk-go-v2 v2.0.0-preview.4+incompatible
	github.com/bmatcuk/doublestar v1.3.0
	github.com/docker/go-units v0.4.0
	github.com/drone-runners/drone-runner-docker v1.3.0
	github.com/drone/drone-admit-members v0.0.0-20190919174107-a33af96e5c11
	github.com/drone/drone-amazon-secrets v0.0.0-20200422210854-c8b51df994f6
	github.com/drone/drone-gc v1.0.0
	github.com/drone/drone-go v1.2.1-0.20200326064413-195394da1018
	github.com/drone/runner-go v1.6.1-0.20200506182602-d2e6327ade15
	github.com/drone/signal v1.0.0
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/golang/mock v1.4.3
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/google/go-github/v28 v28.1.1
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af // indirect
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/rs/zerolog v1.18.0
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v3 v3.0.0-20200506231410-2ff61e1afc86
)
