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

package filename

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fishfinal/fsx"
)

// DefaultEnvironmentServerRunningMode default environment.
const (
	DefaultEnvironmentServerRunningMode = "SERVER_RUNNING_MODE"
)

// Extension configure file extension datatype definition.
type Extension string

// String implement Stringer interface.
func (c Extension) String() string {
	return string(c)
}

// Extension configure file extension related constant definition.
const (
	ExtensionOfToml Extension = "toml"
	ExtensionOfYaml Extension = "yaml"
)

// ServerRunningMode application mode type definition.
type ServerRunningMode string

// String implement Stringer interface.
func (a ServerRunningMode) String() string {
	return string(a)
}

// ServerRunningMode related constant definition.
const (
	ServerRunningModeOfTest     ServerRunningMode = "test"
	ServerRunningModeOfDevelop  ServerRunningMode = "develop"
	ServerRunningModeOfRelease  ServerRunningMode = "release"
	ServerRunningModeOfProduct  ServerRunningMode = "product"
	ServerRunningModeOfWorkflow ServerRunningMode = "workflow"
)

const (
	DefaultConfigureFilenamePrefix = "config"
)

type Option interface {
	apply(*option)
}

type ofn func(filename *option)

func (f ofn) apply(filename *option) {
	f(filename)
}

type option struct {
	prefix            string
	extension         Extension
	serverRunningMode *ServerRunningMode
}

func (o *option) AppModeVerify() error {
	if o.serverRunningMode != nil {
		var modes = []string{
			ServerRunningModeOfTest.String(),
			ServerRunningModeOfDevelop.String(),
			ServerRunningModeOfRelease.String(),
			ServerRunningModeOfProduct.String(),
			ServerRunningModeOfWorkflow.String(),
		}

		var exists bool
		for _, cur := range modes {
			if cur == o.serverRunningMode.String() {
				exists = true
			}
		}

		if !exists {
			return fmt.Errorf("invalid ServerRunningMode for %q, the valid ServerRunningMode must in %q",
				*o.serverRunningMode, fmt.Sprintf("[%s]", strings.Join(modes, ",")))
		}
	}

	return nil
}

func newOptionsDefault() *option {
	// Get the default server running mode
	var mode = func() *ServerRunningMode {
		mode := ServerRunningMode(os.Getenv(DefaultEnvironmentServerRunningMode))
		if mode != "" {
			return &mode
		}
		return nil
	}

	o := &option{
		prefix:            DefaultConfigureFilenamePrefix,
		extension:         ExtensionOfYaml,
		serverRunningMode: mode(),
	}

	return o
}

// WithServerRunningMode specifies the server running mode, used to construct the dynamic config filename,
// e.g. WithServerRunningMode(ServerRunningModeOfDevelop) constructs redbull.develop.yaml.
// If it is not a valid ServerRunningMode type or the conversion to ServerRunningMode fails, the option will not take effect.
func WithServerRunningMode(mode any) Option {
	return ofn(func(filename *option) {
		var m *ServerRunningMode
		if gm, ok := mode.(ServerRunningMode); ok {
			m = &gm
		}
		if gm, ok := mode.(string); ok {
			runningMode := ServerRunningMode(gm)
			m = &runningMode
		}

		if m != nil {
			filename.serverRunningMode = m
		}
	})
}

// WithConfigurationPrefix specifies the prefix of the config file name, e.g. if the filename is redbull.test.yaml,
// then we specify WithConfigurationPrefix("redbull").
func WithConfigurationPrefix(prefix string) Option {
	return ofn(func(filename *option) {
		filename.prefix = prefix
	})
}

// WithConfigurationExtension specifies the extension of the config file name, e.g. if the filename is redbull.test.yaml,
// then we specify WithConfigurationExtension("yaml").
func WithConfigurationExtension(extension Extension) Option {
	return ofn(func(filename *option) {
		filename.extension = extension
	})
}

// Filename config option.
type Filename string

// String convert Filename to string type.
func (c Filename) String() string {
	return string(c)
}

// GetFilename get config file basename.
func GetFilename(path string, options ...Option) string {
	ops := newOptionsDefault()
	for _, o := range options {
		o.apply(ops)
	}

	err := ops.AppModeVerify()
	if err != nil {
		panic(any(err))
	}

	var basename = func(ops *option) string {
		if ops.serverRunningMode != nil {
			return fmt.Sprintf("%s.%s", ops.prefix, ops.serverRunningMode)
		}
		return ops.prefix
	}

	var name = basename(ops)
	filename := fmt.Sprintf("%s.%s", name, ops.extension)
	absolute, err := filepath.Abs(path)
	if err != nil {
		configPath := fmt.Sprintf("[%s]", path)
		panic(fmt.Sprintf("Cannot get the absolute path of the config file %q from %q",
			filename, configPath))
	}

	if fsx.IsNotExists(filepath.Join(absolute, filename)) {
		panic(fmt.Sprintf("Cannot find the application config file %q in the %q directory",
			filename, filepath.Dir(absolute)))
	}

	return filepath.Join(path, filename)
}
