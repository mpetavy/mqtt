@echo off

rem clean tls directory
if exist tls\ rd /s /q tls
mkdir tls

rem Generate private key for the root CA
openssl genpkey -algorithm RSA -out tls\ca.key
rem Generate self-signed root CA certificate
openssl req -x509 -new -nodes -key tls\ca.key -sha256 -days 3650 -out tls\ca.crt -subj "/CN=ca" -addext "subjectAltName=DNS:ca"

rem Generate private key for the server
openssl genpkey -algorithm RSA -out tls\bridge.key
rem Generate a CSR for the server
openssl req -new -key tls\bridge.key -out tls\bridge.csr -subj "/CN=bridge" -addext "subjectAltName=DNS:bridge"
rem Sign the server CSR with the root CA
openssl x509 -req -in tls\bridge.csr -CA tls\ca.crt -CAkey tls\ca.key -CAcreateserial -out tls\bridge.crt -days 365

rem Generate private key for the server
openssl genpkey -algorithm RSA -out tls\broker.key
rem Generate a CSR for the server
openssl req -new -key tls\broker.key -out tls\broker.csr -subj "/CN=bridge" -addext "subjectAltName=DNS:bridge"
rem Sign the server CSR with the root CA
openssl x509 -req -in tls\broker.csr -CA tls\ca.crt -CAkey tls\ca.key -CAcreateserial -out tls\broker.crt -days 365

rem Verify the server certificate against the root CA
openssl verify -CAfile tls\ca.crt tls\bridge.crt

rem Verify the server certificate against the root CA
openssl verify -CAfile tls\ca.crt tls\broker.crt

rem copy the certs
copy tls\ca.* bridge
copy tls\bridge.* bridge

rem copy the certs
copy tls\ca.* broker
copy tls\broker.* broker
