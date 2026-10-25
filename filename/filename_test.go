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
	"os"
	"path/filepath"
	"testing"

	"github.com/fishfinal/fsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestFilenameSuite(t *testing.T) {
	suite.Run(t, &FilenameSuite{})
}

type FilenameSuite struct {
	suite.Suite

	baseDir     string
	testDataDir string
}

func (suite *FilenameSuite) AfterTest(suiteName, testName string) {
	os.Clearenv()
}

func (suite *FilenameSuite) BeforeTest(suiteName, testName string) {
	suite.baseDir = "./../"
	path := fsx.FileSystemPath(filepath.Join("test", "etc"))
	suite.testDataDir = fsx.MustGetFilePathWithFileSystemPath(suite.baseDir, fsx.WithSpecificFileSystemPath(path))
}

func (suite *FilenameSuite) Test_GetConfigFilename() {
	actual := GetFilename(suite.testDataDir)
	assert.Contains(suite.T(), actual, "config.yaml")
}

func (suite *FilenameSuite) Test_GetConfigFilename_WithEnvironment() {
	// unit test case: product
	err := os.Setenv(DefaultEnvironmentServerRunningMode, "test")
	assert.NoError(suite.T(), err)

	actual := GetFilename(suite.testDataDir)
	assert.Contains(suite.T(), actual, "config.test.yaml")

	actual = GetFilename(suite.testDataDir, WithServerRunningMode(ServerRunningModeOfDevelop))
	assert.Contains(suite.T(), actual, "config.develop.yaml")

	// unit test case: invalid run mode
	err = os.Setenv(DefaultEnvironmentServerRunningMode, "testing")
	assert.NoError(suite.T(), err)
	suite.Panicsf(func() { GetFilename(suite.testDataDir) }, "error message %s", "formatted")
}

func (suite *FilenameSuite) Test_GetConfigFilename_WithEnvironmentForTest() {
	actual := GetFilename(suite.testDataDir, WithConfigurationPrefix("conf"))
	assert.Contains(suite.T(), actual, "conf.yaml")

	actual = GetFilename(suite.testDataDir, WithServerRunningMode(ServerRunningModeOfDevelop), WithConfigurationPrefix("conf"))
	assert.Contains(suite.T(), actual, "conf.develop.yaml")
}
