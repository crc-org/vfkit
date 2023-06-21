package config

import (
	"encoding/json"
	"fmt"
)

// The technique for json (de)serialization was explained here:
// http://gregtrowbridge.com/golang-json-serialization-with-interfaces/

type vmComponentKind string

const (
	// Bootloader kinds
	efiBootloader   vmComponentKind = "efiBootloader"
	linuxBootloader vmComponentKind = "linuxBootloader"

	// VirtIO device kinds
	vfNet    vmComponentKind = "virtionet"
	vfVsock  vmComponentKind = "virtiosock"
	vfBlk    vmComponentKind = "virtioblk"
	vfFs     vmComponentKind = "virtiofs"
	vfRng    vmComponentKind = "virtiorng"
	vfSerial vmComponentKind = "virtioserial"
	vfGpu    vmComponentKind = "virtiogpu"
	vfInput  vmComponentKind = "virtioinput"
)

func unmarshalBootloader(rawMsg json.RawMessage) (Bootloader, error) {
	var (
		kind       string
		blmap      map[string]*json.RawMessage
		bootloader Bootloader
	)
	if err := json.Unmarshal(rawMsg, &blmap); err != nil {
		return nil, err
	}

	rawKind := blmap["kind"]
	if rawKind == nil {
		return nil, fmt.Errorf("missing 'kind' node")
	}
	if err := json.Unmarshal(*rawKind, &kind); err != nil {
		return nil, err
	}
	delete(blmap, "kind")
	b, err := json.Marshal(blmap)
	if err != nil {
		return nil, err
	}
	switch kind {
	case string(efiBootloader):
		var efi EFIBootloader
		err = json.Unmarshal(b, &efi)
		if err == nil {
			bootloader = &efi
		}
	case string(linuxBootloader):
		var linux LinuxBootloader
		err = json.Unmarshal(b, &linux)
		if err == nil {
			bootloader = &linux
		}
	default:
		return nil, fmt.Errorf("unknown 'kind' field: '%s'", kind)
	}

	return bootloader, nil
}

// UnmarshalJSON is a custom deserializer for VirtualMachine.  The custom work
// is needed because VirtualMachine uses interfaces in its struct and JSON cannot
// determine which implementation of the interface to deserialize to.
func (vm *VirtualMachine) UnmarshalJSON(b []byte) error {
	var (
		err   error
		input map[string]*json.RawMessage
	)

	if err := json.Unmarshal(b, &input); err != nil {
		return err
	}

	for idx, rawMsg := range input {
		if rawMsg == nil {
			continue
		}
		switch idx {
		case "vcpus":
			err = json.Unmarshal(*rawMsg, &vm.Vcpus)
		case "memoryBytes":
			err = json.Unmarshal(*rawMsg, &vm.MemoryBytes)
		case "bootloader":
			var bootloader Bootloader
			bootloader, err = unmarshalBootloader(*rawMsg)
			if err == nil {
				vm.Bootloader = bootloader
			}
		case "timesync":
			err = json.Unmarshal(*rawMsg, &vm.Timesync)
		case "devices":
			var (
				devices []*json.RawMessage
				dmap    map[string]*json.RawMessage
				kind    string
			)

			err = json.Unmarshal(*rawMsg, &devices)

			for _, msg := range devices {
				if err := json.Unmarshal(*msg, &dmap); err != nil {
					return err
				}
				rawKind := dmap["kind"]
				if rawKind == nil {
					return fmt.Errorf("missing 'kind' node")
				}
				if err := json.Unmarshal(*rawKind, &kind); err != nil {
					return err
				}
				delete(dmap, "kind")
				b, err := json.Marshal(dmap)
				if err != nil {
					return err
				}
				switch kind {
				case string(vfNet):
					var newDevice VirtioNet
					err = json.Unmarshal(b, &newDevice)
					if err == nil {
						vm.Devices = append(vm.Devices, &newDevice)
					}
				case string(vfVsock):
					var newDevice VirtioVsock
					err = json.Unmarshal(b, &newDevice)
					if err == nil {
						vm.Devices = append(vm.Devices, &newDevice)
					}
				case string(vfBlk):
					var newDevice VirtioBlk
					err = json.Unmarshal(b, &newDevice)
					if err == nil {
						vm.Devices = append(vm.Devices, &newDevice)
					}
				case string(vfFs):
					var newDevice VirtioFs
					err = json.Unmarshal(b, &newDevice)
					if err == nil {
						vm.Devices = append(vm.Devices, &newDevice)
					}
				case string(vfRng):
					var newDevice VirtioRng
					err = json.Unmarshal(b, &newDevice)
					if err == nil {
						vm.Devices = append(vm.Devices, &newDevice)
					}
				case string(vfSerial):
					var newDevice VirtioSerial
					err = json.Unmarshal(b, &newDevice)
					if err == nil {
						vm.Devices = append(vm.Devices, &newDevice)
					}
				case string(vfGpu):
					var newDevice VirtioGPU
					err = json.Unmarshal(b, &newDevice)
					if err == nil {
						vm.Devices = append(vm.Devices, &newDevice)
					}
				case string(vfInput):
					var newDevice VirtioInput
					err = json.Unmarshal(b, &newDevice)
					if err == nil {
						vm.Devices = append(vm.Devices, &newDevice)
					}
				default:
					err = fmt.Errorf("unknown 'kind' field: '%s'", kind)
				}
			} // end for-loop for devices

		} // end switch

		if err != nil {
			return err
		}
	} // end for-loop
	return nil
}

func (bootloader *EFIBootloader) MarshalJSON() ([]byte, error) {
	type blWithKind struct {
		Kind vmComponentKind `json:"kind"`
		EFIBootloader
	}
	return json.Marshal(blWithKind{
		Kind:          efiBootloader,
		EFIBootloader: *bootloader,
	})
}

func (bootloader *LinuxBootloader) MarshalJSON() ([]byte, error) {
	type blWithKind struct {
		Kind vmComponentKind `json:"kind"`
		LinuxBootloader
	}
	return json.Marshal(blWithKind{
		Kind:            linuxBootloader,
		LinuxBootloader: *bootloader,
	})
}

func (dev *VirtioNet) MarshalJSON() ([]byte, error) {
	type devWithKind struct {
		Kind vmComponentKind `json:"kind"`
		VirtioNet
	}
	return json.Marshal(devWithKind{
		Kind:      vfNet,
		VirtioNet: *dev,
	})
}

func (dev *VirtioVsock) MarshalJSON() ([]byte, error) {
	type devWithKind struct {
		Kind vmComponentKind `json:"kind"`
		VirtioVsock
	}
	return json.Marshal(devWithKind{
		Kind:        vfVsock,
		VirtioVsock: *dev,
	})
}

func (dev *VirtioBlk) MarshalJSON() ([]byte, error) {
	type devWithKind struct {
		Kind vmComponentKind `json:"kind"`
		VirtioBlk
	}
	return json.Marshal(devWithKind{
		Kind:      vfBlk,
		VirtioBlk: *dev,
	})
}

func (dev *VirtioFs) MarshalJSON() ([]byte, error) {
	type devWithKind struct {
		Kind vmComponentKind `json:"kind"`
		VirtioFs
	}
	return json.Marshal(devWithKind{
		Kind:     vfFs,
		VirtioFs: *dev,
	})
}

func (dev *VirtioRng) MarshalJSON() ([]byte, error) {
	type devWithKind struct {
		Kind vmComponentKind `json:"kind"`
		VirtioRng
	}
	return json.Marshal(devWithKind{
		Kind:      vfRng,
		VirtioRng: *dev,
	})
}

func (dev *VirtioSerial) MarshalJSON() ([]byte, error) {
	type devWithKind struct {
		Kind vmComponentKind `json:"kind"`
		VirtioSerial
	}
	return json.Marshal(devWithKind{
		Kind:         vfSerial,
		VirtioSerial: *dev,
	})
}

func (dev *VirtioGPU) MarshalJSON() ([]byte, error) {
	type devWithKind struct {
		Kind vmComponentKind `json:"kind"`
		VirtioGPU
	}
	return json.Marshal(devWithKind{
		Kind:      vfGpu,
		VirtioGPU: *dev,
	})
}

func (dev *VirtioInput) MarshalJSON() ([]byte, error) {
	type devWithKind struct {
		Kind vmComponentKind `json:"kind"`
		VirtioInput
	}
	return json.Marshal(devWithKind{
		Kind:        vfInput,
		VirtioInput: *dev,
	})
}
