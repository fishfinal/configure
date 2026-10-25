// Copyright 2021 fishfinal <contact.fishfinal@gmail.com>;
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configure

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Custom log instance.
var logger = log.New(os.Stdout, "[CONFIGURE] ", log.Ldate|log.Ltime|log.LUTC)

// Latest processed watcher callback time.
var latestProcessed time.Time

// Debounce interval.
const debounceInterval = 500 * time.Millisecond

// ErrInvalidConfigurationParserRecipient invalid configuration parser recipient.
var ErrInvalidConfigurationParserRecipient = fmt.Errorf("configuration parser recipient must be specified")

// Option interface implementation Option mode.
type Option interface {
	apply(*option)
}

// option
type option struct {
	// Specifies the structure type used to map file
	// configuration to a concrete structure(must be a pointer type).
	receiver interface{}

	// The file type can be any type supported by Viper.
	// Such as `toml/yaml/yml/json` and so on. Default by toml.
	configType string

	// Config file name, notice contains not file extension,
	// configType field is the specified extension.
	// Specific file names such as `config`. Default by config.
	configName string

	// Specific configure file path.
	// Such as:
	//		test/data
	//		config
	// The default is the directory where the program runs
	configPath string

	// Configuring change callback function list.
	callbacks []func(interface{}, interface{})

	// verbose by default false.
	verbose bool
}

// option function use implementation Option interface.
type ofc func(*option)

// apply the specified option configuration field.
func (o ofc) apply(option *option) {
	o(option)
}

// WithFilename Specifies the relative or absolute path to the configuration file.
func WithFilename(filename string) Option {
	return ofc(func(option *option) {
		base := filepath.Base(filename)
		ext := strings.TrimPrefix(filepath.Ext(filename), ".")
		dir := filepath.Dir(filename)

		option.configPath = dir
		option.configName = strings.TrimSuffix(base, filepath.Ext(filename))
		option.configType = ext
	})
}

// WithReceiver Specifies the configuration structure (must be a pointer).
func WithReceiver(receiver interface{}) Option {
	return ofc(func(option *option) {
		option.receiver = receiver
	})
}

// WithCallback Add configuration change callback function.
//
// ---
//
//	Note:
//	This option is only effective when specified in the NewWatcher constructor.
//	If specified in the NewConfigure constructor, a warning similar to the following will be produced:
//	[WARNING] [CONFIGURE] 2025/11/09 08:19:23 Ignore specifies callbacks
func WithCallback(callback func(interface{}, interface{})) Option {
	return ofc(func(option *option) {
		option.callbacks = append(option.callbacks, callback)
	})
}

// WithVerbose specifies whether to enable verbose log output.
func WithVerbose(verbose bool) Option {
	return ofc(func(option *option) {
		option.verbose = verbose
	})
}

// Watcher configuration watcher.
type Watcher struct {
	config interface{}
	mutex  sync.RWMutex
}

// GetConfigure safely retrieves the current configuration.
func (w *Watcher) GetConfigure() interface{} {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.config
}

// NewWatcher Real-time read config file configure into the Configure struct.
// example:
//
//	type Config struct {
//	    Server struct {
//	        Port int `mapstructure:"port"`
//	    } `mapstructure:"server"`
//	}
//
//	// Each callback function has a single responsibility
//	func updateDatabasePool(outdated, newConfig interface{}) {
//	    // Only update the database connection pool
//	}
//
//	func adjustLogLevel(outdated, newConfig interface{}) {
//	    // Only adjust the log level
//	}
//
//	func reloadTLSCertificates(outdated, newConfig interface{}) {
//	    // Only reload TLS certificates
//	}
//
//	func notifyConfigChange(outdated, newConfig interface{}) {
//	    // Only send configuration change notifications
//	    if v, ok := newConfig.(*Config); ok {
//	    	fmt.Printf("Config updated! New port: %d\n", v.Server.Port)
//	    }
//	}
//
//	// Monitor server port changes
//	func serverPortConfigChange(outdated, newConfig interface{}) {
//	    if v, ok := newConfig.(*Config); ok {
//	    	fmt.Printf("Config updated! New port: %d\n", v.Server.Port)
//	    }
//	}
//
//
//	func main() {
//	    watcher, err := configure.NewConfigure(
//	        configure.WithFilename("config.yaml"),
//	        configure.WithReceiver(&Config{}),
//	        configure.WithCallback(updateDatabasePool),    // database connection pool update
//	        configure.WithCallback(adjustLogLevel),        // log level adjustment
//	        configure.WithCallback(reloadTLSCertificates), // TLS certificate reload
//	        configure.WithCallback(notifyConfigChange),    // notify other services
//	    )
//
//	    config := watcher.GetConfigure()
//	    // ...
//	}
func NewWatcher(options ...Option) (*Watcher, error) {
	var ops = option{
		configType: "yaml",
		configName: "config",
		configPath: ".",
	}
	for _, o := range options {
		o.apply(&ops)
	}

	if ops.receiver == nil {
		return nil, ErrInvalidConfigurationParserRecipient
	}

	err := exists(ops)
	if err != nil {
		return nil, err
	}

	viper.SetConfigType(ops.configType)
	viper.SetConfigName(ops.configName)
	viper.AddConfigPath(ops.configPath)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.WatchConfig()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// Initially parse the configuration
	var watcher = &Watcher{config: ops.receiver}
	if err := viper.Unmarshal(watcher.config); err != nil {
		return nil, err
	}

	// Set configuration change callback
	viper.OnConfigChange(func(event fsnotify.Event) {
		watcher.mutex.Lock()
		defer watcher.mutex.Unlock()

		// Debounce processing
		if time.Since(latestProcessed) < debounceInterval {
			return
		}
		latestProcessed = time.Now()

		// Handle write events and create events (Kubernetes ConfigMap updates trigger CREATE).
		// However, the following handling does not achieve the ultimate goal of a general-purpose package,
		// so it is deprecated; instead, whether the final callback handling is needed is determined
		// only by comparing the configuration contents.
		//if !event.Op.Has(fsnotify.Write) && !event.Op.Has(fsnotify.Create) {
		//	logger.Printf("Watcher receive event type %q, ignore", event.Op.String())
		//	return
		//}

		// Deep copy current config into the outdated.
		marshal, err := json.Marshal(watcher.config)
		if err != nil {
			logger.Printf("Error marshaling current config: %v\n", err)
			return
		}

		// Get the type of the current configuration (the type pointed to by the pointer)
		t := reflect.TypeOf(watcher.config).Elem()
		newPtr := reflect.New(t).Interface()

		// Parse the data into the new type
		err = json.Unmarshal(marshal, newPtr)
		if err != nil {
			logger.Printf("Error unmarshalling to outdated config: %v\n", err)
			return
		}

		outdated := newPtr

		if err := viper.Unmarshal(watcher.config); err != nil {
			logger.Printf("Error unmarshal configuration: %v\n", err)
			return
		}

		// Check whether the configuration has actually changed
		o, _ := json.Marshal(outdated)
		n, _ := json.Marshal(watcher.config)
		if string(o) == string(n) {
			logger.Printf("Watcher receive event type %q, but configuration unchanged, skipping callbacks\n", event.Op.String())
			return
		}

		logger.Printf("Watcher receive event type %q, allow %v, processing configuration change",
			event.Op.String(), event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Create))

		// Execute all registered callback functions
		for _, callback := range ops.callbacks {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Printf("Callback panic: %v\n", r)
					}
				}()
				callback(outdated, watcher.config)
			}()
		}
	})
	viper.WatchConfig()

	return watcher, nil
}

func exists(ops option) error {
	var err error
	var temporary = filepath.Join(ops.configPath, fmt.Sprintf("%s.%s", ops.configName, ops.configType))
	_, err = os.Stat(temporary)
	if err != nil {
		return err
	}

	return nil
}

// NewConfigure Read config file configure into the Configure struct.
func NewConfigure(options ...Option) (interface{}, error) {
	var ops = option{
		configType: "yaml",
		configName: "config",
		configPath: ".",
	}
	for _, o := range options {
		o.apply(&ops)
	}

	if ops.receiver == nil {
		return nil, ErrInvalidConfigurationParserRecipient
	}

	err := exists(ops)
	if err != nil {
		return nil, err
	}

	if ops.callbacks != nil {
		outdated := logger.Prefix()
		logger.SetPrefix(fmt.Sprintf("[WARNING] %s", outdated))
		logger.Printf("Ignore specifies callbacks\n")
		logger.SetPrefix(outdated)
	}

	viper.SetConfigType(ops.configType)
	viper.SetConfigName(ops.configName)

	// Support

	viper.AddConfigPath(ops.configPath)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(&ops.receiver); err != nil {
		return nil, err
	}

	return ops.receiver, nil
}
