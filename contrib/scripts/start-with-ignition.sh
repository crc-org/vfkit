#!/bin/sh

# This script can be used to start a vfkit VM
# with ignition configuration using the SSH key provided by the user.
# It expects SSH_PUB_KEY to be provided by the user.
# SSH_USER can be provided to set the name of the user. The default user is test.
# These values are used to generate an ignition file which
# is then passed to vfkit using the --ignition flag.
# The $DISK_IMG variable needs to be set by the user to a
# valid image path for the VM.
#
# Once the VM is running, the user can connect to it using their
# provided key. The VM IP can be found in `/var/db/dhcpd_leases`
# by searching for the VM MAC address (72:20:43:d4:38:62)
#
# Example:
# $ SSH_USER=test DISK_IMG=out/fedora-coreos-41.20250302.3.2-applehv.aarch64.raw \
#   SSH_PUB_KEY=id_ed25519.pub \
#   ./contrib/scripts/start-with-ignition.sh
#
# $ ssh -i id_ed25519 test@192.168.64.14

set -exu

SSH_USER=${SSH_USER:-"test"}

if [ ! -f "$SSH_PUB_KEY" ]; then
  echo "Error: '$SSH_PUB_KEY' does not exist"
  exit 1
fi

if [ ! -f "$DISK_IMG" ]; then
  echo "Error: '$DISK_IMG' does not exist"
  exit 1
fi

PUBLIC_KEY=$(cat "$SSH_PUB_KEY")

mkdir -p out

cat <<EOF > out/config.ign
{
  "ignition": {
    "version": "3.3.0"
  },
  "passwd": {
    "users": [
      {
        "name": "${SSH_USER}",
        "sshAuthorizedKeys": [
          "${PUBLIC_KEY}"
        ]
      }
    ]
  }
}
EOF

./out/vfkit --cpus 2 --memory 2048 \
    --ignition out/config.ign \
    --bootloader efi,variable-store=out/efistore.nvram,create \
    --device virtio-blk,path="$DISK_IMG" \
    --device virtio-serial,logFilePath=out/ignition-vfkit.log \
    --device virtio-net,nat,mac=72:20:43:d4:38:62 \
    --device virtio-rng
