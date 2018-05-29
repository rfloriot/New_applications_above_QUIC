#!/usr/bin/env bash

dd if=/dev/urandom of=4k bs=4k count=1
dd if=/dev/urandom of=8k bs=8k count=1
dd if=/dev/urandom of=16k bs=16k count=1
dd if=/dev/urandom of=32k bs=32k count=1
dd if=/dev/urandom of=64k bs=64k count=1
dd if=/dev/urandom of=128k bs=128k count=1
dd if=/dev/urandom of=256k bs=256k count=1
dd if=/dev/urandom of=512k bs=512k count=1
dd if=/dev/urandom of=1024k bs=1024k count=1
dd if=/dev/urandom of=2048k bs=2048k count=1
dd if=/dev/urandom of=4096k bs=4096k count=1
dd if=/dev/urandom of=8192k bs=8192k count=1