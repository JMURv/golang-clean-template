## Configuration

### GitHub Actions
- Specify **Docker Hub** `USERNAME`, `PASSWORD` and desired `IMAGE_NAME` secrets in GH Actions repo

This is required to build and push docker image

### App
Configuration files placed in `/configs/{local|dev|prod}.config.yaml`
Example file looks like that:

```yaml
serviceName: "svc-name"

server: 
  mode: "dev" # dev, prod
  port: 8080
  scheme: "http"
  domain: "localhost"

db:
  host: "localhost"
  port: 5432
  user: "app_owner"
  password: "app_password"
  database: "app_db"

redis:
  addr: "localhost:6379"
  pass: ""

jaeger:
  sampler:
    type: "const"
    param: 1
  reporter:
    LogSpans: true
    LocalAgentHostPort: "localhost:6831"
```

- Create your own `local.config.yaml` based on `example.config.yaml`
- Create your own `dev.config.yaml` (it is used in dev docker compose file)
- Create your own `prod.config.yaml` (it is used in prod)

### ENV
Docker compose files using `.env.dev` and `.env.prod` files located at `build/compose/env/` folder, so you need to create them
- Specify `DEV_ENV_FILE` and `PROD_ENV_FILE` vars in `build/Taskfile.yaml`

## Build

### Locally

In root folder run:

```shell
go build -o bin/main ./cmd/main.go
```

After that, you can run app via `./bin/main`

___

### Docker

Head to the `build` folder via:

```shell
cd build
```

After that, you can just start docker compose file that will build image automatically via:

```shell
task dc-dev
```

But if you need to build it manually run:

```shell
task dc-dev-build
```

## Run

### Locally

```shell
go run cmd/main.go
```

Or if you previously build the app, run it via:

```shell
go run bin/main
```

___

### Docker-Compose

Head to the `build` folder via:

```shell
cd build
```

Run dev:

```shell
task dc-dev
```

Run prod:

```shell
task dc-prod
```

___

### K8s

Apply manifests

```shell
task k-up
```

Shutdown manifests

```shell
task k-down
```
