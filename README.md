# Master-Slave Remote Management System

A distributed system for remote desktop management that allows controlling multiple slave machines from a central master interface. The system enables remote administration of connected computers including commands execution and desktop wallpaper management.

## Features

- **Automatic Registration**: Slave machines auto-register with the master server
- **Real-time Connection Monitoring**: Master server tracks online/offline status of slaves
- **Remote Command Execution**: Send commands to slave machines
- **Desktop Customization**: Remotely change wallpapers on slave machines
- **Web Interface**: Easy-to-use UI for controlling remote machines
- **WebSocket Communication**: Real-time updates for connected machines

## Architecture

The system uses a client-server architecture with:

1. **Master Server**: Central control server with web interface
2. **Slave Process**: Client application that runs on target machines
3. **Web UI**: Browser-based interface for administration

## Requirements

- Go 1.15+
- Gorilla WebSocket library
- Windows OS for wallpaper functionality

## Installation

### Master Server

1. Clone this repository
2. Navigate to the master directory
3. Install dependencies:
   ```
   go get github.com/gorilla/websocket
   ```
4. Build the master server:
   ```
   go build -o master.exe main.go
   ```

### Slave Client

1. Navigate to the Slave directory
2. Install dependencies:
   ```
   go get golang.org/x/sys/windows
   ```
3. Build the slave client:
   ```
   go build -o slave.exe slave.go
   ```

## Configuration

### Master Server
- Default web interface port: 8082
- Default registration port: 9999
- Web interface files location: `web/` directory

### Slave Client
- Default slave listening port: 8081
- Default master address: 127.0.0.1:9999 (modify in slave.go for different networks)

## Usage

### Starting the Master Server

1. Run the master executable:
   ```
   ./master.exe
   ```
2. Access the web interface at `http://localhost:8082`

### Starting a Slave Client

1. Run the slave executable on the target machine:
   ```
   ./slave.exe
   ```
2. The slave will automatically register with the master and appear in the web interface

### Supported Commands

The slave client supports the following commands:

- `ping`: Check if the slave is online
- `exit`: Terminate the slave process
- `setbg:<path>`: Change the desktop wallpaper (Windows only)

## Web Interface

The web interface provides:

- List of connected slave machines
- Controls to send commands to selected machines
- Wallpaper change functionality
- Real-time status updates

## Network Architecture

1. Master server listens on port 9999 for slave registrations
2. Slaves connect to master on startup to register their presence
3. Master periodically checks slave status with ping commands
4. Web clients connect to master via WebSocket for real-time updates

## Security Considerations

This system is designed for internal networks and trusted environments. Consider the following security enhancements for production use:

- Implement authentication for the web interface
- Add encryption for communication between master and slaves
- Limit allowed commands based on user permissions
- Use TLS for web connections

## Troubleshooting

- If slaves don't appear in the web interface, check network connectivity
- Ensure firewall allows connections on ports 8081, 8082, and 9999
- Check logs for registration errors
- If wallpaper changes fail, verify file path accessibility on slave machines

## License

This project is provided as-is with no warranty. Use at your own risk.

## Future Improvements

- Support for Linux/macOS systems
- File transfer capabilities
- Remote shell access
- Group command execution
- Task scheduling
- Authentication system
