#!/bin/sh
set -eu

: "${S3_BUCKET:=minimart}"

if aws s3api head-bucket --bucket "$S3_BUCKET" 2>/dev/null; then
  exit 0
fi

aws s3 mb "s3://$S3_BUCKET"