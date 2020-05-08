An extension for drone with a built-in garbage collector _Please note this project requires Drone server version 1.4 or higher._

## Installation

Create a shared secret:

```console
$ openssl rand -hex 16
bea26a2221fd8090ea38720fc445eca6
```

Download and run the plugin:

```console
$ docker run -d \
  --publish=3000:3000 \
  --env=DRONE_DEBUG=true \
  --env=DRONE_SECRET=bea26a2221fd8090ea38720fc445eca6 \
  --env=DRONE_GITHUB_TOKEN=$DRONE_GITHUB_TOKEN \
  --env=DRONE_GITHUB_ORG=$DRONE_GITHUB_ORG \
  --env=DRONE_GITHUB_TEAM=$DRONE_GITHUB_TEAM \
  --restart=always \
  --name=infrastructure-plugin andrewstucki/drone-infrastructure-plugin
```

Update your Drone server configuration to include the plugin address and the shared secret.

```text
DRONE_CONVERT_PLUGIN_ENDPOINT=http://1.2.3.4:3000/convert
DRONE_CONVERT_PLUGIN_SECRET=bea26a2221fd8090ea38720fc445eca6
DRONE_ADMISSION_PLUGIN_ENDPOINT=http://1.2.3.4:3000/admit
DRONE_ADMISSION_PLUGIN_SECRET=bea26a2221fd8090ea38720fc445eca6
```
