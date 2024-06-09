#!/bin/sh

set -x
set -e

[ -d tls ] && rm -rf tls
[ ! -d tls ] && mkdir tls

# Generate private key for the root CA
openssl genpkey -algorithm RSA -out tls/ca.key
# Generate self-signed root CA certificate
openssl req -x509 -new -nodes -key tls/ca.key -sha256 -days 3650 -out tls/ca.crt -subj "/CN=ca" -addext "subjectAltName=DNS:ca"

# Generate private key for the server
openssl genpkey -algorithm RSA -out tls/bridge.key
# Generate a CSR for the server
openssl req -new -key tls/bridge.key -out tls/bridge.csr -subj "/CN=bridge" -addext "subjectAltName=DNS:bridge"
# Sign the server CSR with the root CA
openssl x509 -req -in tls/bridge.csr -CA tls/ca.crt -CAkey tls/ca.key -CAcreateserial -out tls/bridge.crt -days 365

# Generate private key for the server
openssl genpkey -algorithm RSA -out tls/broker.key
# Generate a CSR for the server
openssl req -new -key tls/broker.key -out tls/broker.csr -subj "/CN=broker" -addext "subjectAltName=DNS:broker"
# Sign the server CSR with the root CA
openssl x509 -req -in tls/broker.csr -CA tls/ca.crt -CAkey tls/ca.key -CAcreateserial -out tls/broker.crt -days 365

# Verify the server certificate against the root CA
openssl verify -CAfile tls/ca.crt tls/bridge.crt

# Verify the server certificate against the root CA
openssl verify -CAfile tls/ca.crt tls/broker.crt
#
## cp the certs
#cp tls/ca.crt bridge
#cp tls/bridge.key bridge
#cp tls/bridge.crt bridge
#
## cp the certs
#cp tls/ca.crt broker
#cp tls/broker.key broker
#cp tls/broker.crt broker
