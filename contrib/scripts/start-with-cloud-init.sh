#!/bin/sh

# This script can be used to start a vfkit VM
# with cloud-init configuration using the SSH key provided by the user.
# It expects SSH_PUB_KEY to be provided by the user.
# SSH_USER can be set to provide the name of the user. The default value is test.
# These values are used to generate a cloud-init user-data file.
# The HOST_NAME variable can be set by the user to provide
# metadata (instance id and name) about the VM. If not set, default
# name of the VM will be set to vfkit-vm.
# These values are then passed to vfkit using the --cloud-init flag.
# The $DISK_IMG variable needs to be set by the user to a
# valid image path for the VM.
#
# The script has optional support for network-config. The user
# can specify a network-config file using $NETWORK_CONFIG variable.
#
# Once the VM is running, the user can connect to it using their
# provided key. The VM IP can be found in `/var/db/dhcpd_leases`
# by searching for the HOST_NAME or MAC address (72:20:43:d4:38:62).
#
# Example:
# $ SSH_USER=test HOST_NAME=vm1 DISK_IMG=Fedora-Cloud-Base-AmazonEC2-41-1.4.aarch64.raw \
# SSH_PUB_KEY=id_rsa.pub NETWORK_CONFIG=network-config ./contrib/scripts/start-with-cloud-init.sh
#
# $ ssh -i id_rsa test@192.168.64.14

set -exu

HOST_NAME=${HOST_NAME:-"vfkit-vm"}
SSH_USER=${SSH_USER:-"test"}
NETWORK_CONFIG=${NETWORK_CONFIG:-}

if [ ! -f "$SSH_PUB_KEY" ]; then
  echo "Error: '$SSH_PUB_KEY' does not exist"
  exit 1
fi

if [ ! -f "$DISK_IMG" ]; then
  echo "Error: '$DISK_IMG' does not exist"
  exit 1
fi

if [ -n "$NETWORK_CONFIG" ] && [ ! -f "$NETWORK_CONFIG" ]; then
  echo "Error: '$NETWORK_CONFIG' does not exist"
  exit 1
fi

PUBLIC_KEY=$(cat "$SSH_PUB_KEY")

mkdir -p out

cat <<EOF > out/user-data
#cloud-config
users:
  - name: $SSH_USER
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: users
    shell: /bin/bash
    lock_passwd: true
    ssh_authorized_keys:
      - "$PUBLIC_KEY"
ssh_pwauth: false
chpasswd:
  expire: false
EOF

cat <<EOF > out/meta-data
instance-id: $HOST_NAME
local-hostname: $HOST_NAME
EOF

CLOUD_INIT_ARGS="out/user-data,out/meta-data"
if [ -n "$NETWORK_CONFIG" ]; then
  CLOUD_INIT_ARGS="$CLOUD_INIT_ARGS,$NETWORK_CONFIG"
fi

./out/vfkit --cpus 2 --memory 2048 \
    --bootloader efi,variable-store=out/efistore.nvram,create \
    --cloud-init "$CLOUD_INIT_ARGS" \
    --device virtio-blk,path="$DISK_IMG" \
    --device virtio-serial,logFilePath=out/cloud-init.log \
    --device virtio-net,nat,mac=72:20:43:d4:38:62 \
    --device virtio-rng
