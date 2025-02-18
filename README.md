## Configuration

### GitHub Actions
- Specify **Docker Hub** `USERNAME`, `PASSWORD` and desired `IMAGE_NAME` secrets in GH Actions repo

This is required to build and push docker image

### App
Configuration files placed in `/configs/{local|dev|prod}.config.yaml`
Example file looks like that:

```yaml
serviceName: "svc-name"
secret: "DYHlaJpPiZ"

server:
  mode: "dev"
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
    CollectorEndpoint: "http://localhost:14268/api/traces"
```

- Create your own `local.config.yaml` based on `example.config.yaml`
- Create your own `dev.config.yaml` (it is used in dev docker compose file)
- Create your own `prod.config.yaml` (it is used in prod)

### ENV
Docker compose files using `.env.dev` and `.env.prod` files located at `build/compose/env/` folder, so you need to create them

## Build
### Locally

In root folder run (uses `local.config.yaml`):

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

## Run
### Locally

```shell
go run cmd/main.go
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

Also there is ability to up svcs like: `prometheus`, `jaeger`, `node-exporter`, `grafana`:
```shell
task dc-observe
```
Services are available at:

| Сервис     | Адрес                  |
|------------|------------------------|
| App        | http://localhost:8080  |
| Prometheus | http://localhost:9090  |
| Jaeger     | http://localhost:16686 |
| Grafana    | http://localhost:3000  |

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

## Tests
Spin up testing DB for `E2E` tests:
```shell
docker run -d --name pg-container \
  -e POSTGRES_USER=app_owner \
  -e POSTGRES_PASSWORD=app_password \
  -e POSTGRES_DB=app_db_test \
  -p 5432:5432 \
  -v pgdata:/var/lib/postgresql/data \
  postgres:15.0-alpine
```
After that, run `task t-integ`
