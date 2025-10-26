# Prism API Authentication

This document describes the authentication mechanism for the Prism API.

## Overview

Prism API uses a simple API key authentication scheme. When enabled, all API requests must include the API key in the `X-API-Key` header. This provides basic security for your Prism deployments.

## Key Features

- **Opt-in Authentication**: By default, authentication is disabled for backward compatibility
- **Simple API Key**: A single API key is used for all API requests
- **Secure Key Generation**: Keys are generated using cryptographically secure random values
- **Revocation Support**: Keys can be revoked at any time
- **No Expiration**: Keys don't expire until revoked (to avoid disruption)

## Enabling Authentication

Authentication is enabled by generating an API key:

```bash
prism auth generate
```

This will generate a new API key and store it securely. The key will be displayed once and must be saved by the user.

## API Endpoints

### Generate API Key

```
POST /api/v1/auth
```

Generates a new API key. If a key already exists, it will be replaced.

**Response:**
```json
{
  "api_key": "5f9a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a",
  "created_at": "2023-06-15T10:30:00Z",
  "message": "API key generated successfully. This key will not be shown again."
}
```

### Get Authentication Status

```
GET /api/v1/auth
```

**Headers:**
- `X-API-Key`: Your API key (if authentication is enabled)

**Response:**
```json
{
  "auth_enabled": true,
  "authenticated": true,
  "created_at": "2023-06-15T10:30:00Z"
}
```

### Revoke API Key

```
DELETE /api/v1/auth
```

**Headers:**
- `X-API-Key`: Your API key

**Response:**
- `204 No Content` - Key successfully revoked

## Using Authentication in API Requests

Once authentication is enabled, all API requests must include the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/instances
```

### Client Configuration

The Prism CLI automatically manages API keys. When using the API directly, you need to:

1. Generate an API key
2. Store the key securely
3. Include the key in all API requests

### Error Responses

When authentication fails, the API returns:

```json
{
  "code": 401,
  "message": "API key required"
}
```

or

```json
{
  "code": 401,
  "message": "Invalid API key"
}
```

## Security Considerations

- API keys should be treated as sensitive credentials
- Store API keys securely and don't expose them in scripts or environment variables
- Use HTTPS when deploying the daemon on a network
- Consider implementing additional security measures for production deployments
- Authentication is simple by design for ease of use - it's primarily designed for personal use or trusted environments

## Special Endpoints

The following endpoints don't require authentication:

- `/api/v1/ping` - Health check endpoint
- `/api/v1/auth` (POST only) - For generating the initial API key

## Best Practices

1. **Store keys securely**: Don't expose API keys in scripts or config files
2. **Use the CLI**: The `cws` CLI handles key management automatically
3. **Revoke compromised keys**: Immediately revoke keys that may have been exposed
4. **Use HTTPS**: When deploying the daemon on a network, use HTTPS to encrypt traffic
5. **Rotate keys periodically**: Generate new keys regularly for best security

## CLI Usage

```bash
# Generate a new API key
prism auth generate

# Check authentication status
prism auth status

# Revoke the current API key
prism auth revoke
```