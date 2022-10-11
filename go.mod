module github.com/crc-org/vfkit

go 1.17

require (
	github.com/Code-Hex/vz/v2 v2.2.1-0.20221008022127-1b0b4ea5fd24
	github.com/docker/go-units v0.4.0
	github.com/h2non/filetype v1.1.3
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	golang.org/x/sys v0.0.0-20221010170243-090e33056c14
	inet.af/tcpproxy v0.0.0-20210824174053-2e577fef49e2
)

require (
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)

replace github.com/Code-Hex/vz/v2 => github.com/cfergeau/vz/v2 v2.0.0-20221012132510-7c18af23f09f
