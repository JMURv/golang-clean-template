### REMOVE ME
Find and replace all `app-template` to your project's name.
As well, do it for `github.com/JMURv/golang-clean-template`, that used in go backend.

### GitHub Actions
Specify **Docker Hub** `USERNAME`, `PASSWORD` and desired `IMAGE_NAME` secrets in GH Actions repo.

This is required to build and push docker image.
### END

## Stack
|          | Technology |
|----------|------------|
| Backend  | Golang     |

---
## Configuration
### App
Configuration files for local dev placed in `/config`

Configuration files for docker dev placed in `/build/configs/envs`
- Create your own `.env` based on `.env.example`:
```shell
cp config/.env.example config/.env && \
cp build/configs/envs/.env.example build/configs/envs/.env.dev && \
cp build/configs/envs/.env.example build/configs/envs/.env.prod
```

## Build
### Locally
In root folder run:
```shell
go build -o bin/main ./cmd/main.go
```
After that, you can run app via `./bin/main`
___

### Docker
In the root folder run:
```shell
task dc-dev-build
```

or run:
```shell
task dc-prod-build
```

To build `dev` or `prod` containers respectively

## Run
### Locally
```shell
go run cmd/main.go
```

___

### Docker Compose
Run dev (requires `build/configs/envs/.env.dev`):
```shell
task dc-dev
```

Run prod (requires `build/configs/envs/.env.prod`):
```shell
task dc-prod
```

Also, there is ability to up svcs like: `prometheus`, `jaeger`, `node-exporter`, `grafana`.

You can include necessary svcs manually in your desired `compose*.yaml` file from `compose-base.yaml`.

Services are available at:

| Сервис     | Адрес                  |
|------------|------------------------|
| App (HTTP) | http://localhost:8080  |
| App (GRPC) | http://localhost:50050 |
| Prometheus | http://localhost:9090  |
| Jaeger     | http://localhost:16686 |
| Grafana    | http://localhost:3000  |

___

### K8s

- Create your own `configMap` and `secretMap` based on examples:
```shell
cp build/k8s/cfg/cfg.example.yaml build/k8s/cfg/cfg.yaml && \
cp build/k8s/cfg/secret.example.yaml build/k8s/cfg/secret.yaml 
```

Apply manifests:
```shell
task k-up
```

Shutdown manifests:
```shell
task k-down
```

## Tests
### Integration
Run:
```shell
task t-integration
```
It will spin up all containers for integration testing automatically using `testcontainers`.