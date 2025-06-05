#!/bin/sh

# SPDX-FileCopyrightText: The vfkit authors
# SPDX-License-Identifier: Apache-2.0
#
# This script can be used to start a raw disk image with vfkit using gvproxy
# for usermode networking.
# The mac address must be 5a:94:ef:e4:0c:ee as this is the address expected by gvproxy.
#
# After the VM is running, its ssh port is reachable on port 2223 on localhost (127.0.0.1).
#
# If the path to `--listen-vfkit` is too long (more than ~100 characters), then
# gvproxy/vfkit will fail to start as a unix socket filename must be less than
# that.
#
# This script creates an overlay for the disk image, the disk image is not modified.
#
set -exuo pipefail

: "${GVPROXY:=gvproxy}"
: "${VFKIT:=./out/vfkit}"

DISK_IMAGE="${1?Usage: $0 diskimage}"
VM_NAME="$(basename ${DISK_IMAGE})"

${GVPROXY} --mtu 1500 --ssh-port 2223 --listen-vfkit unixgram://$(pwd)/${VM_NAME}.sock --log-file ${VM_NAME}.gvproxy.log --pid-file ${VM_NAME}.gvproxy.pid &

TO_REMOVE="${VM_NAME}.sock ${VM_NAME}.gvproxy.pid ${VM_NAME}.overlay.img ${VM_NAME}.efistore.nvram"
trap 'if [[ -f "${VM_NAME}.gvproxy.pid" ]]; then kill $(cat ${VM_NAME}.gvproxy.pid); fi; rm -f ${TO_REMOVE}' EXIT

cp -c ${DISK_IMAGE} "${VM_NAME}".overlay.img

${VFKIT} --cpus 2 --memory 2048 \
    --bootloader efi,variable-store=${VM_NAME}.efistore.nvram,create \
    --device virtio-blk,path=${VM_NAME}.overlay.img \
    --device virtio-serial,logFilePath=${VM_NAME}.log \
    --device virtio-net,unixSocketPath=$(pwd)/${VM_NAME}.sock,mac=5a:94:ef:e4:0c:ee \
    --device virtio-rng
