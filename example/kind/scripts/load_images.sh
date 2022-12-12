#!/bin/bash

kind load docker-image liqo/liqo-webhook:latest --name $1;
kind load docker-image liqo/auth-service:v0.6.0 --name $1;
kind load docker-image liqo/liqo-controller-manager:v0.6.0 --name $1;
kind load docker-image liqo/crd-replicator:v0.6.0 --name $1;
kind load docker-image liqo/liqonet:v0.6.0 --name $1;
kind load docker-image liqo/metric-agent:v0.6.0 --name $1;
kind load docker-image liqo/uninstaller:v0.6.0 --name $1;
kind load docker-image envoyproxy/envoy:v1.21.0 --name $1
