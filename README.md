# `introspect`

[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/vasu1124/introspect/badge)](https://securityscorecards.dev/viewer/?uri=github.com/vasu1124/introspect)
![Build](https://github.com/vasu1124/introspect/workflows/Build/badge.svg) 
[![GitHub issues](https://img.shields.io/github/issues/vasu1124/introspect.svg)](https://img.shields.io/github/issues/vasu1124/introspect/issues)
[![Go Report Card](https://goreportcard.com/badge/github.com/vasu1124/introspect)](https://goreportcard.com/report/github.com/vasu1124/introspect)
[![Docker Pulls](https://img.shields.io/docker/pulls/vasu1124/introspect.svg?maxAge=2592000)](https://hub.docker.com/r/vasu1124/introspect/)

## 🎯 What is Introspect?

**Introspect** is a go-based web application designed to demonstrate and explore Kubernetes capabilities, showcaseing a few cloud-native patterns, features, and integrations. It provides a modern web interface with multiple interactive features and serves a teaching tool for containerized applications running in Kubernetes environments.

## ✨ Features
- **Kubernetes Runtime Environment**: Inspect environment variables, configuration, and runtime information
- **Database Integrations**: Work with MongoDB, etcd, and Valkey (Redis-compatible)
- **Kubernetes Operators**: Custom Resource Definitions (CRDs) and controller patterns
- **Leadership Election**: Distributed coordination using Kubernetes leases
- **Admission Webhooks**: Validation webhook implementation
- **Observability**: Prometheus metrics, health checks, and logging
- **Cookie Management**: Session handling and cookie inspection
- **HPA Simulation**: Workload is generated (with fractals from the Mandelbrot set) for scaling demonstration with HPA.

## 🛠️ Prerequisites

To develop and test Introspect locally, you'll need the following tools installed on your laptop:

### Required Tools

1. **Go** (1.25.4 or later)
2. **kubectl** (for interacting with Kubernetes clusters)
3. **Docker** (for building container images)
4. **Kubernetes Cluster** (e.g. Kind)
5. **Tilt** (for local development workflow)
6. **Make** (for build automation)

   ```bash
   # Check version
   go version
   kubectl version --client
   docker --version
   kind version
   tilt version
   make --version
   ```

### Optional Tools

- **Helm** (for managing dependencies like MongoDB, etcd)
- **cfssl** (for TLS certificate generation)
- **direnv** (for environment variable management)

## 🚀 Local Development with Tilt

Tilt provides a fast, iterative development experience for Kubernetes applications. It automatically rebuilds, deploys, and updates your application as you make changes.

### Quick Start

1. **Create a Kubernetes Cluster**:
   ```bash
   # Using Kind
   make kind-up
   
   # Or manually
   kind create cluster --config kind.yaml --wait 5m
   ```

2. **Start Tilt**:
   ```bash
   tilt up
   ```

3. **Access the Tilt UI**:
   - Open your browser to http://localhost:10350
   - The Tilt UI shows build status, logs, and resource health

4. **Access the Application**:
   - Once deployed, the application is available at http://localhost:9090
   - Tilt automatically port-forwards the service

5. **Clean up**:
   - Stop the running `tilt up` server with crlt-c. Then run `tilt down` to clean up resources:
   ```bash
   tilt down
   ```

## 📁 Project Structure

```
introspect/
├── cmd/                          # Application entry points
│   ├── main.go                   # Main entry point
│   └── introspect/               # CLI commands
│       ├── root.go               # Root command and config
│       └── server.go             # Server command
│
├── pkg/                          # Application packages
│   ├── ...                       # Various demos
│   ├── server/                   # HTTP server
│   └── version/                  # Version information
│
├── tmpl/                         # HTML templates
│   ├── layout.html               # Base layout template
│   ├── ...                       # Various demo templates
│
├── css/                          # Stylesheets
│   └── ...
│
├── kubernetes/                   # Kubernetes manifests
├── hack/                         # Development scripts
├── demo/                         # Demo scripts and examples
│
├── .github/                      # GitHub workflows
│
├── Tiltfile                      # Tilt configuration
├── Makefile                      # Build automation
├── Dockerfile                    # Main Dockerfile
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
└── README.md                     # This file
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## 📄 License

This project is licensed under the GPL-3.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with:
- [Kubernetes](https://kubernetes.io/)
- [Go](https://golang.org/)
- [Tilt](https://tilt.dev/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Cobra](https://github.com/spf13/cobra)
- and many, many more
