# How to test different claims using the static token

The static token's payload is located under `config/static-token-payload.json`.
It defines a sample org_admin with full administrative rights on the private API of the fleet manager.

The token can be used seamlessly by simply adding it in bearer format to API requests. It will allow access to all APIs.

It's additionally also used within the fleetshard synchronizer for developing without requiring sso.redhat.com.

In case you want to change the static token, follow the steps below (if you do not require to completely re-create the 
JWKS files, please re-use the ones under `dev/static-token` respectively):

1. (optional): You have to create JSON web keypair. For simplicity, you can use [mkjwk.org](http://mkjwk.org/) to generate the keypair.
2. (optional): Use the following options to generate your keypair:
```
Key Size:   2048
Key Use:    Signature
Alogirhtm:  RS256 RSASSA-PKCS1-v1_5 using SHA-256
Key ID:     SHA-256
Show X.509: Yes
```
3. (optional): Copy the values of the `Public Key, Public Key (X.509 PEM Format), Private Key (X.509 PEM Format)` fields.
4. (optional): Replace the values within `dev/static-token/jwks-priv.key.pem|jwks-pub-key.pem|jwks-pub-key.json` respectively with the previously copied values.
5. (optional): Copy the value of the `dev/static-token/jwks-pub-key.json` and append the value to the array within `config/jwks-file-static.json`.
6. Open [jwt.io](https://jwt.io), and paste the value of `dev/static-token/jwks-pub-key.json` into the `HEADER` field in the decoded column.
7. Copy the values of both `dev/static-token/jwks-pub-key.pem|jwks-priv-key.pem` respectively, pasting them into the `VERIFY SIGNATURE` fields.
8. Copy the payload data contained within `config/static-token-payload.json` and adjust the payload to your liking.
9. Once finished:
   1. copy the encoded JWT and update the value within `fleetshard/pkg/fleetmanager/static-token`.
   2. copy the payload data and update the value within `config/static-token-payload.json`.

If you have re-created the JWKS files, ensure that fleet manager is re-started with the new values of the `config/jwks-file-static.json`.
This also includes any staging instances.

Locally, you have the option to explicitly set the JWKS file to be used by the fleet manager using the following flag:
```shell
./fleet-manager --jwks-file path/to/your/file
```
