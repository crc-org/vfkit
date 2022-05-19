module github.com/code-ready/vfkit

go 1.16

require (
	github.com/Code-Hex/vz v0.0.5-0.20220406150231-a2ebc854a261
	github.com/docker/go-units v0.4.0
	github.com/h2non/filetype v1.1.3
	github.com/rs/xid v1.4.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	golang.org/x/sys v0.0.0-20220519141025-dcacdad47464
	inet.af/tcpproxy v0.0.0-20210824174053-2e577fef49e2
)

replace github.com/Code-Hex/vz => github.com/cfergeau/vz v0.0.5-0.20220629154958-9aad23fce70e
