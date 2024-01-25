#!/bin/bash

[ ! -d tls/ca.key ] && mkdir tls

[ -f tls/ca.key ] && rm tls/ca.key
[ -f tls/ca.crt ] && rm tls/ca.crt
[ -f tls/server.csr ] && rm tls/server.csr
[ -f tls/server.key ] && rm tls/server.key
[ -f tls/server.crt ] && rm tls/server.crt

set -e

# Generate private key for the root CA
openssl genpkey -algorithm RSA -out tls/ca.key

# Generate self-signed root CA certificate
openssl req -x509 -new -nodes -key tls/ca.key -sha256 -days 3650 -out tls/ca.crt -subj "/CN=ca"

# Generate private key for the server
openssl genpkey -algorithm RSA -out tls/server.key

# Generate a CSR for the server
openssl req -new -key tls/server.key -out tls/server.csr -subj "/CN=bridge" -addext "subjectAltName=DNS:bridge"

# Sign the server CSR with the root CA
openssl x509 -req -in tls/server.csr -CA tls/ca.crt -CAkey tls/ca.key -CAcreateserial -out tls/server.crt -days 365

# Verify the server certificate against the root CA
openssl verify -CAfile tls/ca.crt tls/server.crt
