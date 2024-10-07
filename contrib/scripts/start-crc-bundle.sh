#!/bin/sh

# This script can be used to start a
# [CRC/OpenShift Local bundle](https://crc.dev/blog/)
# with vfkit.
# It expects the bundle to be unpacked in ~/.crc/cache.
# It creates an overlay for the disk image, the files in ~/.crc/cache are not
# modified.
#
# Once the VM is running, you can connect to it using the `id_*_crc` SSH key
# in the bundle directory. The default user is `core`.
# The VM IP can be found in `/var/db/dhcpd_leases` by searching for the VM MAC
# address (72:20:43:d4:38:62)
#
# Example:
# $ sh contrib/scripts/start-crc-bundle.sh ~/.crc/cache/crc_microshift_vfkit_4.16.7/
# $ ssh -i ~/.crc/cache/crc_microshift_vfkit_4.16.7/id_ecdsa_crc core@192.168.64.2

set -exu

YQ=${YQ:-yq}
BUNDLE_PATH=$1
DISKIMG=$(cat ${BUNDLE_PATH}/crc-bundle-info.json | ${YQ} .storage.diskImages[0].name)
cp -c ${BUNDLE_PATH}/${DISKIMG} overlay.img

./out/vfkit --cpus 2 --memory 2048 \
    --bootloader efi,variable-store=efistore.nvram,create \
    --device virtio-blk,path=overlay.img \
    --device virtio-serial,logFilePath=start-bundle.log \
    --device virtio-net,nat,mac=72:20:43:d4:38:62 \
    --device virtio-rng
