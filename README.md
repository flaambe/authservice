# Authentication service
Simple authentication service with **Go** and **MongoDB**

## Build

```bash
make build
```
## Test

```bash
make test
```

## API

#### /auth
* `POST` : Get access and refresh tokens pair

#### /refreshToken
* `POST` : Refresh access and refresh tokens pair

#### /deleteToken
* `POST` : Delete specific refresh token

#### /deleteAllTokens
* `POST` : Delete all refresh tokens for specific user

## Usage
Get access and refresh tokens pair

    curl -i -d '{"guid":${GUID}}' -X POST http://localhost:8080/auth

Refresh access and refresh tokens pair

    curl -i -H "Authorization: Bearer ${ACCESS_TOKEN}" -d '{"refresh_token":${REFRESH_TOKEN}}' -X POST http://localhost:8080/refreshToken

Delete specific refresh token

    curl -i -H "Authorization: Bearer ${ACCESS_TOKEN}" -d '{"refresh_token":${REFRESH_TOKEN}}' -d '{"guid":${GUID}}' -X POST http://localhost:8080/deleteToken

Delete all refresh tokens for specific user

    curl -i -H "Authorization: Bearer ${ACCESS_TOKEN}" -X POST http://localhost:8080/deleteAllTokens