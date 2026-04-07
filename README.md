# CS2 Profile Stats API

Intended to be used with [CS2 Profile Stats extension](https://github.com/CS2-Profile-Stats/cs2-profilestats-extension), but can be used for basically anything.

## Installation
- Install [GoLang](https://go.dev/doc/install).
- Clone the repository: `git clone https://github.com/CS2-Profile-Stats/cs2-profilestats-api.git`
- Build: `go build -o api ./cmd/api`

## Usage
- Rename `.env.example` to `.env` and enter your api keys

Can be used as is with `./api` or with [Docker](README.md#docker).  
Runs on port :8080 by default.

### Docker
Use [Docker Compose](https://docs.docker.com/compose/install/linux/): `docker compose up -d --build`
