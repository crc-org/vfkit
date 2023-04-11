#!/bin/sh

set -exu

YQ=${YQ:-yq}
BUNDLE_PATH=$1
KERNEL=$(cat ${BUNDLE_PATH}/crc-bundle-info.json | ${YQ} .nodes[0].kernel)
INITRD=$(cat ${BUNDLE_PATH}/crc-bundle-info.json | ${YQ} .nodes[0].initramfs)
CMDLINE=$(cat ${BUNDLE_PATH}/crc-bundle-info.json | ${YQ} .nodes[0].kernelCmdLine)
DISKIMG=$(cat ${BUNDLE_PATH}/crc-bundle-info.json | ${YQ} .storage.diskImages[0].name)
cp -c ${BUNDLE_PATH}/${DISKIMG} overlay.img

./out/vfkit --cpus 2 --memory 2048 \
    --kernel "${BUNDLE_PATH}/${KERNEL}" \
    --initrd "${BUNDLE_PATH}/${INITRD}" \
    --kernel-cmdline "${CMDLINE}" \
    --device virtio-blk,path=overlay.img \
    --device virtio-serial,logFilePath=start-bundle.log \
    --device virtio-net,nat,mac=72:20:43:d4:38:62 \
    --device virtio-rng
