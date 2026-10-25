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
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/fishfinal/configure/filename"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Config map value from config.yaml(or other format) configure.
type Config struct {
	Application struct {
		Name    string
		Version string
	}
	Database struct {
		MYSQL struct {
			Host     string
			Port     int
			Username string
			Password string
			Database string
		}
		Redis struct {
			Host     string
			Port     int
			Password string
			Database int
		}
	}
}

type Swagger struct {
	Title               string
	Version             string
	Schemes             []string
	Consumes            []string
	Produces            []string
	SecurityDefinitions SecurityDefinitions `mapstructure:"security_definitions"`
}

type SecurityToken struct {
	Type string
	Name string
	In   string
}

type SecurityDefinitions struct {
	AppTokenHeader    SecurityToken `mapstructure:"app_token_header"`
	AppTokenQuery     SecurityToken `mapstructure:"app_token_query"`
	ClientTokenHeader SecurityToken `mapstructure:"client_token_header"`
	ClientTokenQuery  SecurityToken `mapstructure:"client_token_query"`
}

func TestConfigureSuite(t *testing.T) {
	suite.Run(t, &ConfigureSuite{})
}

type ConfigureSuite struct {
	suite.Suite
}

// Test_NewConfigure_WithReceiver_Config tests loading the configuration file and specifying
// the receiver struct for configuration parsing. No environment variables are specified,
// so any configuration values not specified in the original config file should be zero values.
func (suite *ConfigureSuite) Test_NewConfigure_WithReceiver_Config() {
	// test/data/config.yaml
	//
	// 	Application:
	// 	 Name: cms
	// 	 Version: v1.2.3
	// 	Database:
	// 	  MySQL:
	// 	    Host: localhost
	// 	    Port: 3306
	// 	    Username: cms
	// 	    Password: # Automatically resolved via environment variables
	// 	    Database: cms
	// 	  Redis:
	// 	    Host: localhost
	// 	    Port: # Automatically resolved via environment variables; viper automatically parses string environment variables into the target data type, e.g. int
	// 	    Password: # Automatically resolved via environment variables
	// 	    Database: 5
	config, err := NewConfigure(WithFilename("test/data/config.yaml"), WithReceiver(&Config{}))
	assert.NotNil(suite.T(), config)
	assert.NoError(suite.T(), err)

	application := config.(*Config).Application
	assert.Equal(suite.T(), "cms", application.Name)
	assert.Equal(suite.T(), "v1.2.3", application.Version)

	database := config.(*Config).Database
	assert.Equal(suite.T(), "localhost", database.MYSQL.Host)
	assert.Equal(suite.T(), 3306, database.MYSQL.Port)
	assert.Equal(suite.T(), "cms", database.MYSQL.Database)
	assert.Empty(suite.T(), database.MYSQL.Password)

	assert.Equal(suite.T(), "localhost", database.Redis.Host)
	assert.Equal(suite.T(), 0, database.Redis.Port)
	assert.Equal(suite.T(), "", database.Redis.Password)
	assert.Equal(suite.T(), 5, database.Redis.Database)
}

// Test_NewConfigure_WithFilename_FileNotExists tests loading a non-existent configuration file.
func (suite *ConfigureSuite) Test_NewConfigure_WithFilename_FileNotExists() {
	config, err := NewConfigure(WithFilename("test/data/not-exists-config.yaml"), WithReceiver(&Config{}))
	assert.Nil(suite.T(), config)
	assert.Error(suite.T(), err)
	log.Printf("err: %v", err)
}

// Test_NewWatcher_WithFilename_FileNotExists tests loading a non-existent configuration file.
func (suite *ConfigureSuite) Test_NewWatcher_WithFilename_FileNotExists() {
	config, err := NewWatcher(WithFilename("test/data/not-exists-config.yaml"), WithReceiver(&Config{}))
	assert.Nil(suite.T(), config)
	assert.Error(suite.T(), err)
	log.Printf("err: %v", err)
}

// Test_NewConfigure_WithReceiver_Config tests loading the configuration file and specifying
// the receiver struct for configuration parsing. With environment variables specified,
// any configuration values not specified in the original config file should be correctly
// injected and resolved through environment variables.
func (suite *ConfigureSuite) Test_NewConfigure_WithReceiver_Config_AutomaticEnv() {
	suite.NoError(os.Setenv("DATABASE_MYSQL_PASSWORD", "12345@PASSWORD@MYSQL"))
	suite.NoError(os.Setenv("DATABASE_REDIS_PORT", "6378"))
	suite.NoError(os.Setenv("DATABASE_REDIS_PASSWORD", "12345@PASSWORD@REDIS"))

	// test/data/config.yaml
	//
	// 	Application:
	// 	 Name: cms
	// 	 Version: v1.2.3
	// 	Database:
	// 	  MySQL:
	// 	    Host: localhost
	// 	    Port: 3306
	// 	    Username: cms
	// 	    Password: # Automatically resolved via environment variables
	// 	    Database: cms
	// 	  Redis:
	// 	    Host: localhost
	// 	    Port: # Automatically resolved via environment variables; viper automatically parses string environment variables into the target data type, e.g. int
	// 	    Password: # Automatically resolved via environment variables
	// 	    Database: 5
	config, err := NewConfigure(WithFilename("test/data/config.yaml"), WithReceiver(&Config{}))
	assert.NotNil(suite.T(), config)
	assert.NoError(suite.T(), err)

	application := config.(*Config).Application
	assert.Equal(suite.T(), "cms", application.Name)
	assert.Equal(suite.T(), "v1.2.3", application.Version)

	database := config.(*Config).Database
	assert.Equal(suite.T(), "localhost", database.MYSQL.Host)
	assert.Equal(suite.T(), 3306, database.MYSQL.Port)
	assert.Equal(suite.T(), "cms", database.MYSQL.Database)
	assert.Equal(suite.T(), "12345@PASSWORD@MYSQL", database.MYSQL.Password)

	assert.Equal(suite.T(), "localhost", database.Redis.Host)
	assert.Equal(suite.T(), 6378, database.Redis.Port)
	assert.Equal(suite.T(), "12345@PASSWORD@REDIS", database.Redis.Password)
	assert.Equal(suite.T(), 5, database.Redis.Database)
}

func (suite *ConfigureSuite) Test_NewConfigure_WithReceiver_Swagger() {
	// test/data/swagger.yaml
	//
	//		# API short description.
	//		title: SWAGGER API DOCUMENTATION.
	//
	//		# API version.
	//		version: v1.2.3
	//
	//		# API schemes.
	//		schemes:
	//		  - http
	//		  - https
	//
	//		# API consumes.
	//		consumes:
	//		  - application/json
	//
	//		# API produces.
	//		produces:
	//		  - application/json
	//
	//		# Security definitions
	//		security_definitions:
	//		  app_token_header:
	//		    type: apiKey
	//		    name: X-Access-Token
	//		    in: header
	//		  app_token_query:
	//		    type: apiKey
	//		    name: token
	//		    in: query
	//		  client_token_header:
	//		    type: apiKey
	//		    name: X-Access-Token
	//		    in: header
	//		  client_token_query:
	//		    type: apiKey
	//		    name: token
	//		    in: query
	config, err := NewConfigure(WithFilename("test/data/swagger.yaml"), WithReceiver(&Swagger{}))
	assert.NotNil(suite.T(), config)
	assert.NoError(suite.T(), err)

	swagger := config.(*Swagger)
	assert.Equal(suite.T(), "X-Access-Token", swagger.SecurityDefinitions.ClientTokenHeader.Name)
}

func (suite *ConfigureSuite) Test_NewConfigure_WithFilename() {
	config, err := NewConfigure(WithFilename("test/data/config.yaml"), WithReceiver(&Config{}))
	instance := config.(*Config)
	assert.NotNil(suite.T(), instance)
	assert.NoError(suite.T(), err)
}

// Test_NewConfigure_UseFilename_BuildDynamicConfigurationFilename determines the current
// running mode through environment variables. Valid values are test, develop, release,
// product, workflow. Through filename.GetFilename, the full path of the configuration
// file, such as test/data/redbull.workflow.yaml, can be assembled. Then, the NewConfigure
// option value filename can be constructed via WithFilename(getFilename).
func (suite *ConfigureSuite) Test_NewConfigure_UseFilename_BuildDynamicConfigurationFilename() {
	err := os.Setenv(filename.DefaultEnvironmentServerRunningMode, filename.ServerRunningModeOfWorkflow.String())
	assert.NoError(suite.T(), err)
	defer os.Clearenv()

	getenv := os.Getenv(filename.DefaultEnvironmentServerRunningMode)
	getFilename := filename.GetFilename("test/data", filename.WithServerRunningMode(getenv), filename.WithConfigurationPrefix("redbull"))

	config, err := NewConfigure(WithFilename(getFilename), WithReceiver(&Config{}))
	instance := config.(*Config)
	assert.NotNil(suite.T(), instance)
	assert.NoError(suite.T(), err)
}

// Test_NewConfigure_WithReceiver_Config tests loading the configuration file and specifying
// the receiver struct for configuration parsing. No environment variables are specified,
// so any configuration values not specified in the original config file should be zero values.
func (suite *ConfigureSuite) Test_NewConfigure_WithCallback_Config() {
	// test/data/config.yaml
	//
	// 	Application:
	// 	 Name: cms
	// 	 Version: v1.2.3
	// 	Database:
	// 	  MySQL:
	// 	    Host: localhost
	// 	    Port: 3306
	// 	    Username: cms
	// 	    Password: # Automatically resolved via environment variables
	// 	    Database: cms
	// 	  Redis:
	// 	    Host: localhost
	// 	    Port: # Automatically resolved via environment variables; viper automatically parses string environment variables into the target data type, e.g. int
	// 	    Password: # Automatically resolved via environment variables
	// 	    Database: 5
	config, err := NewConfigure(WithFilename("test/data/config.yaml"), WithReceiver(&Config{}), WithCallback(func(outdatedConfig interface{}, newConfig interface{}) {
		o, ok := outdatedConfig.(*Config)
		if !ok {
			return
		}

		n, ok := newConfig.(*Config)
		if !ok {
			return
		}

		if o.Application.Version != n.Application.Version {
			fmt.Printf("Application version updated")
		}
	}))
	assert.NotNil(suite.T(), config)
	assert.NoError(suite.T(), err)

	application := config.(*Config).Application
	assert.Equal(suite.T(), "cms", application.Name)
	assert.Equal(suite.T(), "v1.2.3", application.Version)

	database := config.(*Config).Database
	assert.Equal(suite.T(), "localhost", database.MYSQL.Host)
	assert.Equal(suite.T(), 3306, database.MYSQL.Port)
	assert.Equal(suite.T(), "cms", database.MYSQL.Database)
	assert.Empty(suite.T(), database.MYSQL.Password)

	assert.Equal(suite.T(), "localhost", database.Redis.Host)
	assert.Equal(suite.T(), 0, database.Redis.Port)
	assert.Equal(suite.T(), "", database.Redis.Password)
	assert.Equal(suite.T(), 5, database.Redis.Database)
}
