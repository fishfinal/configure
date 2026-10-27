<div align="center">
  <img src="assets/banner/configure.jpg" alt="configure banner" />
</div>

# configure

<div align="center">

<img src="https://img.shields.io/badge/Go-%3E%3D1.22-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" />
<img src="https://img.shields.io/badge/license-Apache--2.0-blue?style=for-the-badge" alt="License" />
<img src="https://img.shields.io/badge/build-passing-brightgreen?style=for-the-badge&logo=githubactions&logoColor=white" alt="Build Status" />
<img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=for-the-badge" alt="PRs Welcome" />

</div>

A clean, efficient, and type-safe Go application configuration management package. Built on top of [Viper](https://github.com/spf13/viper), it provides seamless environment variable integration, dynamic configuration monitoring, and a robust callback mechanism.

## Background

Starting a new project often involves dealing with a multitude of configuration settings. Manually mapping configuration files to structured objects each time is repetitive and time-consuming. This package automates the tedious mapping process, allowing you to bind configurations to your Go structs in minutes with high efficiency and safety.

## Features

- **Multi-format Support**: Supports YAML, TOML, JSON, and other Viper-compatible configuration formats.
- **Environment Variable Integration**: Automatic environment variable binding with smart key name conversion (`.` → `_`).
- **Functional Options**: Clean and extensible configuration using the functional options pattern.
- **Dynamic Monitoring**: Real-time configuration file monitoring with automatic reloading.
- **Debounce & Diffing**: Built-in debounce logic to prevent rapid file write triggers, and state comparison to ensure callbacks only fire on actual changes.
- **Change Callbacks**: Register multiple handler functions to react only to specific configuration modifications.
- **Thread Safety**: Read-write locks ensure safe concurrent access to the configuration instance.

## Installation

```bash
go get -u github.com/fishfinal/configure
```

## Quick Start

### Basic Usage

```go
package main

import (
  "fmt"
  "time"

  "github.com/fishfinal/configure"
)

// Define configuration structure
type Config struct {
  Application struct {
    Name    string
    Version string
  }
  Server struct {
    Port    int
    Timeout time.Duration
  }
  Database struct {
    MYSQL struct {
      Host     string
      Port     int
      Username string
      Password string
      Database string
    }
  }
}

func main() {
  // Load configuration from file
  config, err := configure.NewConfigure(
    configure.WithFilename("config.yaml"), // Specify config file
    configure.WithReceiver(&Config{}),     // Specify target struct
  )
  if err != nil {
    panic(fmt.Sprintf("Failed to load config: %v", err))
  }

  cfg := config.(*Config)
  fmt.Printf("App Name: %s\nVersion: %s\n", cfg.Application.Name, cfg.Application.Version)
}
```

### Dynamic Configuration Monitoring

```go
package main

import (
  "fmt"
  "log"
  "time"

  "github.com/fishfinal/configure"
)

type Config struct {
  Application struct {
    Version string
  }
  Server struct {
    Port int
  }
  Database struct {
    PoolSize int `mapstructure:"pool_size"`
  }
}

func main() {
  var cfg Config

  // Create config watcher with callbacks
  watcher, err := configure.NewWatcher(
    configure.WithFilename("config.yaml"),
    configure.WithReceiver(&cfg),
    configure.WithCallback(onServerConfigChange),
    configure.WithCallback(onDatabaseConfigChange),
    configure.WithVerbose(true), // Enable verbose logging
  )
  if err != nil {
    log.Fatal("Failed to create config watcher:", err)
  }

  // Safely get current configuration
  currentCfg := watcher.GetConfigure().(*Config)
  fmt.Printf("Initial config - Port: %d, DB Pool: %d\n", 
    currentCfg.Server.Port, currentCfg.Database.PoolSize)

  // Block main goroutine
  select {}
}

// Callback: React to Server changes
func onServerConfigChange(outdated, newConfig interface{}) {
  oldCfg, oldOk := outdated.(*Config)
  newCfg, newOk := newConfig.(*Config)
  if !oldOk || !newOk {
    return
  }

  if oldCfg.Server.Port != newCfg.Server.Port {
    fmt.Printf("Server port changed from %d to %d\n", oldCfg.Server.Port, newCfg.Server.Port)
    // Restart server or rebind port here
  }
}

// Callback: React to Database changes
func onDatabaseConfigChange(outdated, newConfig interface{}) {
  oldCfg, oldOk := outdated.(*Config)
  newCfg, newOk := newConfig.(*Config)
  if !oldOk || !newOk {
    return
  }

  if oldCfg.Database.PoolSize != newCfg.Database.PoolSize {
    fmt.Printf("DB pool size changed from %d to %d\n", oldCfg.Database.PoolSize, newCfg.Database.PoolSize)
    // Adjust database connection pool here
  }
}
```

### Configuration File Example (`config.yaml`)

```yaml
Application:
  Name: myapp
  Version: v1.0.0
Server:
  Port: 9000
Database:
  MYSQL:
    Host: localhost
    Port: 3306
    Password: ${DATABASE_PASSWORD} # Will be populated from environment variable
  Redis:
    Host: localhost
    Port: 6379
```

## Core API

### Option Functions

- `WithFilename(filename string)`: Specify configuration file path (relative or absolute).
- `WithReceiver(receiver interface{})`: Specify target struct for configuration mapping.
- `WithCallback(callback func(interface{}, interface{}))`: Register configuration change callback function (only works with `NewWatcher`).
- `WithVerbose(verbose bool)`: Enable/disable verbose logging for the watcher.

### Main Functions

- `NewConfigure(options ...Option) (interface{}, error)`: Load and parse configuration statically.
- `NewWatcher(options ...Option) (*Watcher, error)`: Create a configuration watcher for dynamic hot-reloading.
- `Watcher.GetConfigure() interface{}`: Safely retrieve the current configuration instance.

## Environment Variable Support

The package automatically maps environment variables to configuration fields using the following naming convention: `.` becomes `_` and all letters are uppercase.

```go
// Set environment variables
os.Setenv("DATABASE_MYSQL_PASSWORD", "securepassword123")
os.Setenv("DATABASE_REDIS_PORT", "6378")

// Corresponding fields will be populated automatically
type Config struct {
  Database struct {
    MYSQL struct {
      Password string // ← From DATABASE_MYSQL_PASSWORD
    }
    Redis struct {
      Port int        // ← From DATABASE_REDIS_PORT (auto-converted to int)
    }
  }
}
```

## Dynamic Configuration Monitoring

The `NewWatcher` function provides dynamic configuration monitoring, automatically reloading configuration when files change and triggering registered callbacks.

### Smart Debounce & Diffing

To prevent file system events from firing multiple times during a single save operation (common with editors like Vim or IDEs), `configure` uses:
1. **Debounce**: Ignores file events that occur within 500ms of the last processed event.
2. **Deep Equality Check**: It serializes the old and new configurations to JSON and compares them. If the file was touched but the contents didn't actually change, **callbacks are skipped**.

### Callback Design

Callback functions receive two parameters:
- `outdated`: A deep copy of the configuration before changes.
- `newConfig`: The current configuration after changes.

This allows you to compare specific fields and only execute expensive logic (like restarting a server or reloading TLS certs) when necessary.

## Dynamic Configuration File Names

You can combine this package with the built-in `filename` sub-package to load environment-specific configuration files (e.g., `config.dev.yaml`, `config.prod.yaml`).

```go
import (
  "os"
  "github.com/fishfinal/configure"
  "github.com/fishfinal/configure/filename"
)

func main() {
  // Set running environment
  os.Setenv("SERVER_RUNNING_MODE", "develop")

  // Generate dynamic configuration file path
  runningMode := os.Getenv("SERVER_RUNNING_MODE")
  configFile := filename.GetFilename("config",
    filename.WithServerRunningMode(runningMode),
    filename.WithConfigurationPrefix("myapp"),
  )
  // configFile will be "config/myapp.develop.yaml"

  config, err := configure.NewConfigure(
    configure.WithFilename(configFile),
    configure.WithReceiver(&Config{}),
  )
  // ...
}
```

## Best Practices

1. **Use Struct Tags**: Use `mapstructure` tags if your config keys don't map directly to Go's case-insensitive field names.
2. **Single Responsibility Callbacks**: Write separate callback functions for different subsystems (e.g., one for Database, one for Logger) to keep logic isolated.
3. **Panic Recovery**: The watcher automatically recovers from panics inside your callbacks to prevent the main process from crashing, but you should still write defensive code inside callbacks.
4. **Prefix Environment Variables**: Use unified prefixes (e.g., `APP_`) for environment variables to avoid collisions.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
