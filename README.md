# Gateway API Graph Visualizer

An interactive web-based visualization tool for Kubernetes Gateway API resources. This tool provides a real-time graph view of Gateway API resources and their relationships within a Kubernetes cluster.

## Features

- **Real-time Visualization**: Interactive graph showing Gateway API resources and their relationships
- **Multiple Layout Options**: Force, radial, and hierarchical layouts
- **Resource Details**: Click on nodes to view detailed resource information
- **Auto-refresh**: Automatic updates with WebSocket connection
- **Zoom and Pan**: Navigate large graphs with zoom and pan capabilities
- **Color-coded Resources**: Different colors for different resource types

## Supported Resources

- **GatewayClass**: Cluster-scoped configuration templates (v1)
- **Gateway**: Gateway instances bound to GatewayClasses (v1)
- **HTTPRoute**: HTTP routing rules (v1)
- **ReferenceGrant**: Cross-namespace references (v1beta1)

All resources are from the Gateway API v1.2.1 Standard channel. Note that ReferenceGrant is still in v1beta1 as it has not yet graduated to v1 in this version.

## Prerequisites

- Go 1.21 or later
- Kubernetes cluster with Gateway API CRDs installed
- kubectl configured to access your cluster
- Gateway API v1.2.1 or later

## Installation

### Option 1: Run locally

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd gwapi-graph
   ```

2. Install Go dependencies:
   ```bash
   go mod tidy
   ```

3. Run the application:
   ```bash
   go run main.go
   ```

4. Open your browser and navigate to `http://localhost:8080`

### Option 2: Docker

1. Build the Docker image:
   ```bash
   docker build -t gwapi-graph .
   ```

2. Run the container:
   ```bash
   docker run -p 8080:8080 -v ~/.kube/config:/root/.kube/config gwapi-graph
   ```

3. Open your browser and navigate to `http://localhost:8080`

### Option 3: Kubernetes Deployment

1. Apply the Kubernetes manifests:
   ```bash
   kubectl apply -f k8s/
   ```

2. Access the service via port-forward or ingress:
   ```bash
   kubectl port-forward service/gwapi-graph 8080:8080
   ```

## Configuration

The application automatically discovers your Kubernetes configuration:

1. **In-cluster**: Uses the service account token when running inside a Kubernetes pod
2. **Local**: Uses `~/.kube/config` file

## API Endpoints

- `GET /`: Main visualization interface
- `GET /api/resources`: Returns all Gateway API resources
- `GET /api/graph`: Returns graph data structure
- `GET /api/ws`: WebSocket endpoint for real-time updates

## Graph Layouts

### Force Layout (Default)
- Uses D3.js force simulation
- Nodes repel each other while connected nodes attract
- Good for general topology visualization

### Radial Layout
- Places GatewayClasses at the center
- Gateways in the first ring
- Routes in the outer rings
- Good for understanding hierarchy

### Hierarchical Layout
- Vertical arrangement by resource type
- Shows clear resource hierarchy
- Good for understanding dependencies

## Resource Relationships

The visualizer shows the following relationships:

- **GatewayClass → Gateway**: via `gatewayClassName` field
- **Gateway → HTTPRoute**: via `parentRefs` field in HTTPRoute specifications
- **HTTPRoute → Services**: via `backendRefs` field (when available)
- **ReferenceGrant**: Enables cross-namespace references between resources

## Usage

1. **Navigate**: Use mouse to pan and zoom the graph
2. **Select**: Click on nodes to view detailed information
3. **Refresh**: Use the refresh button or enable auto-refresh
4. **Layout**: Switch between different layout algorithms
5. **Reset**: Reset zoom to fit all nodes

## Development

### Project Structure

```
gwapi-graph/
├── main.go                 # Application entry point
├── internal/
│   ├── api/               # HTTP handlers and WebSocket
│   ├── k8s/               # Kubernetes client wrapper
│   └── types/             # Data structures
├── web/
│   ├── templates/         # HTML templates
│   └── static/           # CSS and JavaScript files
├── k8s/                  # Kubernetes deployment manifests
└── Dockerfile           # Container image definition
```

### Adding New Resource Types

1. Add the resource type to `internal/k8s/client.go`
2. Update the `ResourceCollection` struct in `internal/types/types.go`
3. Add graph building logic in `internal/api/handler.go`
4. Update CSS colors in `web/static/style.css`

### Building

```bash
# Build for current platform
go build -o gwapi-graph main.go

# Build for Linux (for Docker)
GOOS=linux GOARCH=amd64 go build -o gwapi-graph main.go
```

## Troubleshooting

### Common Issues

1. **No resources showing**: Check if Gateway API CRDs are installed in your cluster
2. **Connection refused**: Verify kubectl can connect to your cluster
3. **WebSocket errors**: Check network connectivity and firewall settings

### Debug Mode

Set the Gin mode to debug for more verbose logging:
```bash
export GIN_MODE=debug
go run main.go
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Gateway API](https://gateway-api.sigs.k8s.io/) - The Kubernetes Gateway API specification
- [D3.js](https://d3js.org/) - Data visualization library
- [Gin](https://gin-gonic.com/) - Go web framework 