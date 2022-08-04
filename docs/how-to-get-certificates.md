# How to get certificates for stage and prod environments

## Get a new certificate

- Create a new key and CSR

```
openssl genrsa -out tls.key 4096
openssl req -new -key tls.key -out tls.csr
# Input following informations
# Country Name: US
# State or Province Name: North Carolina
# Locality Name: Raleigh
# Organization Name: Red Hat\, Inc.
# Organizationonal Unit Name: leave blank
# Common Name: *.acs-stage.rhcloud.com
# Email Address: rhacs-eng-ms@redhat.com
```

Create a general service now ticket for requesting a certificate (e.g. stage ticket)
