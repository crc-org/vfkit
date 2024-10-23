## Ignition

Ignition uses a JSON configuration file to define the desired changes. The format of this config is specified in detail [here](https://coreos.github.io/ignition/specs/), and its [MIME type](http://www.iana.org/assignments/media-types/application/vnd.coreos.ignition+json) is registered with IANA.

`myconfig.json` file provides an example of configuration that adds a new `testuser`, creates a new file to `/etc/myapp` with the content listed in the same `files` section and add a systemd unit drop-in to modify the existing service `systemd-journald` and sets its environment variable SYSTEMD_LOG_LEVEL to debug.

### Examples

More examples can be found at https://coreos.github.io/ignition/examples/
