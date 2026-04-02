# Google Cloud Dynamic DNS Client

[![CI](https://github.com/b1tsized/gcp-dynamic-dns/actions/workflows/ci.yml/badge.svg)](https://github.com/b1tsized/gcp-dynamic-dns/actions/workflows/ci.yml)
[![Release](https://github.com/b1tsized/gcp-dynamic-dns/actions/workflows/release.yml/badge.svg)](https://github.com/b1tsized/gcp-dynamic-dns/actions/workflows/release.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/b1tsized/gcp-dynamic-dns)](https://hub.docker.com/r/b1tsized/gcp-dynamic-dns)
[![Docker Image Version](https://img.shields.io/docker/v/b1tsized/gcp-dynamic-dns?sort=semver)](https://hub.docker.com/r/b1tsized/gcp-dynamic-dns)
[![Go Version](https://img.shields.io/github/go-mod/go-version/b1tsized/gcp-dynamic-dns?filename=src%2Fapp%2Fgo.mod)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE.txt)

> **Note**: This is an actively maintained fork of [luontola/gcp-dynamic-dns](https://github.com/luontola/gcp-dynamic-dns) with updated dependencies and modernized GCP SDK.

Automatically sync your public IP address to [Google Cloud DNS](https://cloud.google.com/dns) records. Perfect for home servers, self-hosted services, and dynamic IP environments.

## Features

- **Multiple IP Detection Methods**
  - External web service (default)
  - Local network interface
  - Router via UPnP
- **Flexible Authentication**
  - Environment variable (JSON string)
  - Mounted credentials file
- **Lightweight**
  - Minimal Docker image (~10MB)
  - Single static binary
- **Multi-Architecture**
  - `linux/amd64`
  - `linux/arm64`

---

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
  - [Docker Compose](#docker-compose)
  - [UnRaid](#unraid)
- [Configuration](#configuration)
  - [Authentication](#authentication)
  - [Environment Variables](#environment-variables)
- [GCP Setup](#gcp-setup)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

---

## Quick Start

```bash
docker run -d \
  --name gcp-dynamic-dns \
  --network host \
  --restart always \
  -e DNS_NAMES="example.com." \
  -e GOOGLE_PROJECT="your-project-id" \
  -e GOOGLE_CREDENTIALS_JSON='{"type":"service_account",...}' \
  b1tsized/gcp-dynamic-dns sync
```

---

## Installation

### Docker Compose

```yaml
services:
  dyndns:
    image: b1tsized/gcp-dynamic-dns
    container_name: gcp-dynamic-dns
    command: sync
    network_mode: host
    restart: always
    environment:
      DNS_NAMES: example.com. www.example.com.
      GOOGLE_PROJECT: your-project-123456
      # Option 1: Credentials as JSON string (recommended)
      GOOGLE_CREDENTIALS_JSON: '{"type":"service_account","project_id":"..."}'
      # Option 2: Credentials as file
      # GOOGLE_APPLICATION_CREDENTIALS: /gcp-keys.json
    # volumes:
    #   - ./gcp-keys.json:/gcp-keys.json:ro
```

### UnRaid

<details>
<summary>Click to expand UnRaid installation guide</summary>

#### Adding the Container

1. In UnRaid's **Docker** tab, click **Add Container**
2. Configure:

| Field | Value |
|-------|-------|
| **Name** | `gcp-dynamic-dns` |
| **Repository** | `b1tsized/gcp-dynamic-dns` |
| **Network Type** | `host` |
| **Post Arguments** | `sync` |
| **Restart Policy** | `always` |

#### Environment Variables

| Variable | Value |
|----------|-------|
| `DNS_NAMES` | `yourdomain.com.` (must end with `.`) |
| `GOOGLE_PROJECT` | Your GCP project ID |
| `GOOGLE_CREDENTIALS_JSON` | Your service account JSON |

#### Verifying It Works

Check the container logs. You should see:

```
Using credentials from GOOGLE_CREDENTIALS_JSON environment variable
Current IP: 203.0.113.42
Updated DNS: example.com. A -> 203.0.113.42
```

</details>

---

## Configuration

### Authentication

Two methods are supported for GCP credentials:

#### Option 1: Environment Variable (Recommended)

Pass the entire service account JSON as an environment variable:

```yaml
environment:
  GOOGLE_CREDENTIALS_JSON: '{"type":"service_account","project_id":"...","private_key":"..."}'
```

**Pros**: No file mounts, works great with container orchestrators

#### Option 2: File Mount

Mount the JSON key file into the container:

```yaml
environment:
  GOOGLE_APPLICATION_CREDENTIALS: /gcp-keys.json
volumes:
  - /path/to/credentials.json:/gcp-keys.json:ro
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DNS_NAMES` | Yes | - | Space-separated DNS names (must end with `.`) |
| `GOOGLE_PROJECT` | Yes | - | GCP project ID |
| `GOOGLE_CREDENTIALS_JSON` | * | - | Service account JSON content |
| `GOOGLE_APPLICATION_CREDENTIALS` | * | - | Path to credentials file |
| `MODE` | No | `service` | IP detection mode (see below) |
| `SERVICE_URLS` | No | [see below] | Custom IP detection URLs |
| `INTERFACE_NAME` | No | auto | Network interface name |

\* One of `GOOGLE_CREDENTIALS_JSON` or `GOOGLE_APPLICATION_CREDENTIALS` is required

#### IP Detection Modes

| Mode | Description |
|------|-------------|
| `service` | Query external web services (default) |
| `interface` | Read from local network interface |
| `upnp` | Query router via UPnP protocol |

> **Note**: Modes `interface` and `upnp` require `network_mode: host`

#### Default Service URLs

```
https://ipv4.icanhazip.com/
https://checkip.amazonaws.com/
https://ifconfig.me/ip
https://ipinfo.io/ip
```

Services are queried in round-robin fashion every 5 minutes.

---

## GCP Setup

1. Go to [GCP Console → IAM & Admin → Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts)

2. **Create a service account**
   - Name: `dns-updater`
   - Role: **DNS Administrator** (`roles/dns.admin`)

3. **Create a JSON key**
   - Click on the service account
   - Keys → Add Key → Create new key → JSON
   - Download and save securely

4. **Ensure DNS records exist**
   - Records must already exist in Cloud DNS
   - Must be type `A` records
   - Names must end with `.` (e.g., `example.com.`)

---

## Development

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker](https://www.docker.com/get-started)

### Local Development

```bash
# Run tests
cd src/app
go test -v ./...

# Build binary
go build -o ../../app

# Run locally
DNS_NAMES="example.com." GOOGLE_PROJECT="test" ./app list-ip
```

### Docker Build

```bash
# Build image
docker build -t b1tsized/gcp-dynamic-dns .

# Test container
docker run --rm b1tsized/gcp-dynamic-dns help
```

### CI/CD

| Workflow | Trigger | Action |
|----------|---------|--------|
| **CI** | Push/PR to `main` | Run tests, build |
| **Release** | GitHub Release | Build & push to Docker Hub |

### Creating a Release

1. Go to [Releases](../../releases) → **Create a new release**
2. Create tag: `v2.0.0` (semantic versioning)
3. Click **Publish release**

The workflow automatically builds multi-arch images and pushes:
- `b1tsized/gcp-dynamic-dns:latest`
- `b1tsized/gcp-dynamic-dns:2.0.0`
- `b1tsized/gcp-dynamic-dns:2.0`
- `b1tsized/gcp-dynamic-dns:2`

### Repository Secrets

Required for releases:

| Secret | Description |
|--------|-------------|
| `DOCKERHUB_USERNAME` | Docker Hub username |
| `DOCKERHUB_TOKEN` | [Docker Hub access token](https://hub.docker.com/settings/security) |

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## Changes from Upstream

This fork includes:

- Updated to **Go 1.25**
- Updated all dependencies to latest versions
- Modernized GCP SDK (`dns.NewService()` instead of deprecated `dns.New()`)
- Narrower OAuth scope (`NdevClouddnsReadwriteScope`)
- **New**: `GOOGLE_CREDENTIALS_JSON` environment variable support
- **New**: Multi-architecture Docker images (amd64/arm64)
- **New**: GitHub Actions CI/CD
- Code quality improvements

---

## License

[Apache License 2.0](LICENSE.txt)

Original work Copyright © 2023 [Esko Luontola](https://github.com/luontola)

---

<p align="center">
  <a href="https://github.com/b1tsized/gcp-dynamic-dns/issues">Report Bug</a>
  ·
  <a href="https://github.com/b1tsized/gcp-dynamic-dns/issues">Request Feature</a>
</p>
