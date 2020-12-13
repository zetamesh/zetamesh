#!/usr/bin/env bash

GOOS=linux make
tar -czf zetamesh.tar.gz -C bin zetamesh

scp zetamesh.tar.gz root@$DEPLOY_HOST:/opt/zetamesh/
ssh root@$DEPLOY_HOST << EOF
  cd /opt/zetamesh/
  tar -xzf zetamesh.tar.gz
EOF

rm zetamesh.tar.gz