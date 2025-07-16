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

Run dev with observation containers:
```shell
task dc-dev-obs
```

Run prod (requires `build/configs/envs/.env.prod`):
```shell
task dc-prod
```

Run prod with observation containers:
```shell
task dc-prod-obs
```

Observe profile starts svcs like: `prometheus`, `jaeger`, `node-exporter`, `grafana` and etc.

Services are available at:

| Сервис           | Адрес                  |
|------------------|------------------------|
| App (HTTP)       | http://localhost:8080  |
| App (GRPC)       | http://localhost:50050 |
| App (PROMETHEUS) | http://localhost:8085  |
| Prometheus       | http://localhost:9090  |
| Node-exporter    | http://localhost:9100  |
| Jaeger           | http://localhost:16686 |
| Loki             | http://localhost:3100  |
| Grafana          | http://localhost:3000  |

More information could be found inside `compose.yaml`.

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

### Load testing 
Run:
```shell
task dc-k6
```

It will spin up all `prod` environment, including observation containers and starts k6 scenario.
Grafana has prebuilt dashboard for k6.
