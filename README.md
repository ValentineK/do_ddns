# Digital Ocean Dynamic DNS Updater

A lightweight Go application that automatically updates Digital Ocean DNS records when your public IP address changes. Perfect for home servers, dynamic IP connections, or any scenario where you need to keep DNS records in sync with your current public IP.

## Features

- Automatic IP detection using api.myip.com
- Digital Ocean DNS API integration
- Docker support with multi-stage builds
- Configurable via environment variables
- Runs as non-root user for security
- Built-in scheduling with docker-compose

## Quick Start

### Using Docker Compose (Recommended)

1. Create a `.env` file with your configuration:
```bash
DO_TOKEN=your_digital_ocean_token_here
DOMAIN=example.com
SUBDOMAIN=home
RECORD_TYPE=A
TTL=300
DEBUG=0
```

2. Run with docker-compose:
```bash
docker-compose up -d
```

The application will check and update your DNS record every 300 seconds (5 minutes).

### Using Docker

```bash
docker build -t ddns-updater .
docker run --env-file .env ddns-updater
```

### Direct Go Execution

```bash
go build -o ddns-updater main.go
./ddns-updater
```

## Configuration

All configuration is done via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DO_TOKEN` | Yes | - | Digital Ocean API token |
| `DOMAIN` | Yes | - | Domain to update (e.g., "example.com") |
| `SUBDOMAIN` | Yes | - | Subdomain to update (e.g., "home") |
| `RECORD_TYPE` | No | A | DNS record type |
| `TTL` | No | 300 | Time to live in seconds |
| `DEBUG` | No | 0 | Enable debug logging ("1" or "true") |

## Getting a Digital Ocean API Token

1. Go to the [Digital Ocean Control Panel](https://cloud.digitalocean.com/account/api/tokens)
2. Click "Generate New Token"
3. Give it a name and select "Read" and "Write" permissions
4. Copy the token and use it as your `DO_TOKEN`

## How It Works

1. Fetches your current public IP from api.myip.com
2. Retrieves existing DNS records from Digital Ocean API
3. Finds the specified subdomain record
4. Compares current IP with DNS record IP
5. Updates the DNS record only if the IP has changed

## Example Output

```
Configuration loaded:
  Domain: example.com
  Subdomain: home
  Record Type: A
  TTL: 300
  Token: dop_v1_a***b123

Current IP: 203.0.113.123
IP changed from 203.0.113.100 to 203.0.113.123, updating DNS record...
Successfully updated home.example.com to 203.0.113.123
```

## Security

- Runs as non-root user in Docker container
- Uses HTTPS for all API calls
- Token is partially masked in logs
- Minimal Alpine Linux base image

## License

This project is open source and available under the MIT License.