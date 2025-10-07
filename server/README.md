# Terra Mystica Server

Go backend server for Terra Mystica online multiplayer game.

## Building with Bazel

### Prerequisites
- Bazel installed (you have this already)
- Go 1.21+ (will be downloaded by Bazel if needed)

### Build the server
```bash
bazel build //cmd/server:server
```

### Run the server
```bash
bazel run //cmd/server:server
```

Or run the built binary directly:
```bash
./bazel-bin/cmd/server/server_/server
```

### Development

The server will start on port 8080 with the following endpoints:
- `ws://localhost:8080/ws` - WebSocket endpoint for game connections
- `http://localhost:8080/health` - Health check endpoint

### Project Structure
```
server/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   └── websocket/       # WebSocket hub and client management
├── WORKSPACE            # Bazel workspace configuration
├── BUILD.bazel          # Root build file
├── deps.bzl             # External Go dependencies
├── go.mod               # Go module definition
└── go.sum               # Go module checksums
```

### Testing the WebSocket Connection

You can test the WebSocket connection using a simple client or browser console:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onopen = () => console.log('Connected');
ws.onmessage = (event) => console.log('Received:', event.data);
ws.send('Hello from client!');
```
