package main

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/crc-org/vfkit/pkg/vf"
	sleepnotifier "github.com/prashantgupta24/mac-sleep-notifier/notifier"
	log "github.com/sirupsen/logrus"
)

func syncGuestTime(conn net.Conn) error {
	qemugaCmdTemplate := `{"execute": "guest-set-time", "arguments":{"time": %d}}` + "\n"
	qemugaCmd := fmt.Sprintf(qemugaCmdTemplate, time.Now().UnixNano())

	log.Debugf("sending %s to qemu-guest-agent", qemugaCmd)
	_, err := conn.Write([]byte(qemugaCmd))
	if err != nil {
		return err
	}
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}

	if response != `{"return": {}}`+"\n" {
		return fmt.Errorf("Unexpected response from qemu-guest-agent: %s", response)
	}

	return nil
}

func watchWakeupNotifications(vm *vz.VirtualMachine, vsockPort uint) {
	var vsockConn net.Conn
	defer func() {
		if vsockConn != nil {
			_ = vsockConn.Close()
		}
	}()

	sleepNotifierCh := sleepnotifier.GetInstance().Start()
	for {
		select {
		case activity := <-sleepNotifierCh:
			log.Debugf("Sleep notification: %s", activity)
			if activity.Type == sleepnotifier.Awake {
				log.Infof("machine awake")
				if vsockConn == nil {
					var err error
					vsockConn, err = vf.ConnectVsockSync(vm, vsockPort)
					if err != nil {
						log.Debugf("error connecting to vsock port %d: %v", vsockPort, err)
						break
					}
				}
				if err := syncGuestTime(vsockConn); err != nil {
					log.Debugf("error syncing guest time: %v", err)
				}
			}
		}
	}

}

func setupGuestTimeSync(vm *vz.VirtualMachine, timesync *config.TimeSync) error {
	if timesync == nil {
		return nil
	}

	log.Infof("Setting up host/guest time synchronization")

	go watchWakeupNotifications(vm, timesync.VsockPort())

	return nil
}
