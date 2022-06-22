# How to test different claims locally

1. Create Keys on this site: https://mkjwk.org/. You can use the following options:
```
Key Size: 2048
Key Use: Signature
Algorithm RS256: RSASSA-PKCS1-v1_5 using SHA-256
Key ID SHA-256
```
**be sure to enable: Show X.509**

2. Copy `Public and Private Keypair Set` field and paste it into new file. i.e. `config/jwks-file-test.json`
3. Copy `Public Key` field. Open: https://jwt.io/ and paste the data into `HEADER` field
4. Copy and paste public and private keys **X.509 PEM Format** into: jwt.io related parts in `VERIFY SIGNATURE` field
5. Copy wanted payload into: jwt.io `PAYLOAD:DATA` field. Example:
```
{
    "exp": 1763919488,
    "iat": 1653918588,
    "auth_time": 1653898675,
    "iss": "https://identity.api.stage.openshift.com/auth/realms/rhoas-dinosaur-sre",
    "aud": "cloud-services",
    "typ": "Bearer",
    "azp": "cloud-services",
    "allowed-origins": [
        "http://127.0.0.1:8000",
    ],
    "realm_access": {
        "roles": [
            "authenticated",
            "fleet-manager-admin-read",
            "fleet-manager-admin-write",
            "fleet-manager-admin-full"
        ]
    },
    "scope": "openid",
    "account_number": "123123",
    "is_internal": true,
    "is_active": true,
    "last_name": "The White",
    "preferred_username": "rh-the-gray",
    "type": "User",
    "locale": "en_us",
    "is_org_admin": false,
    "account_id": "909090",
    "idp": "auth.redhat.com",
    "user_id": "101",
    "org_id": "100001",
    "first_name": "Gandalf",
    "email": "gthewhit@redhat.com",
    "username": "rh-the-gray"
}
```
**NOTE**: important parts could be `realm_access.roles`, `iss`, etc.

6. Use generated Token from jwt.io
7. Start application with JWKS file param: `--jwks-file config/jwks-file-test.json`
