# Backend Python Client

Simple Python client for communicating with the Backend service. This client hides all gRPC implementation details and provides a clean interface for Python agent services.

## Installation

### Development Setup
```bash
# Clone and navigate to the client directory
cd backendapi/client/python

# Run development setup (creates venv, installs deps)
./setup_dev.sh

# Activate the virtual environment
source venv/bin/activate

# Generate gRPC client files
./generate.sh
```

### Production Installation
```bash
pip install ./backendapi/client/python
```

## Usage

### Basic Usage
```python
from backendapi import BackendClient

# Create client
client = BackendClient(host="localhost", port=9090)

# Send a reply to a conversation
success = client.send_reply(
    conversation_id="550e8400-e29b-41d4-a716-446655440000",
    message="Hello from Python agent!"
)

if success:
    print("Reply sent successfully!")

# Clean up
client.close()
```

### Context Manager Usage
```python
from backendapi import BackendClient

with BackendClient(host="localhost", port=9090) as client:
    client.send_reply(
        conversation_id="550e8400-e29b-41d4-a716-446655440000", 
        message="Another reply!"
    )
# Automatically closes connection
```

### Error Handling
```python
from backendapi import BackendClient, BackendError, ConnectionError, RequestError

client = BackendClient()

try:
    client.send_reply(conversation_id, message)
except ConnectionError:
    print("Failed to connect to Backend service")
except RequestError as e:
    print(f"Service returned error: {e}")
except BackendError as e:
    print(f"General error: {e}")
```

## Configuration

### Default Settings
- **Host**: `localhost`
- **Port**: `9090` (gRPC port)

### Environment Variables
The client uses the default values above, but you can override them when creating the client:

```python
client = BackendClient(host="backend-service", port=9090)
```

## Development

### Regenerating gRPC Files
If the protobuf definitions change, regenerate the client:

```bash
# Activate virtual environment
source venv/bin/activate

# Regenerate gRPC files
./generate.sh
```

### Package Structure
```
backendapi/
├── __init__.py              # Package exports
├── client.py                # Main client class
├── exceptions.py            # Custom exceptions
└── generated/               # Auto-generated gRPC files
    ├── infralayer_pb2.py      # Protobuf messages
    └── infralayer_pb2_grpc.py # gRPC client stub
```

## Requirements

- Python 3.8+
- grpcio >= 1.50.0
- grpcio-tools >= 1.50.0

## License

Part of the Backend project.