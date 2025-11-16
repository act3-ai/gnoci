#!/bin/bash

remote="$1"
shift

git config lfs.standalonetransferagent oci
git config lfs.customtransfer.oci.path "git-lfs-remote-oci"
git config lfs.customtransfer.oci.args ""
git config lfs.customtransfer.oci.batch true
git config lfs.customtransfer.oci.concurrent false
git config lfs.url "oci://$remote"