package state

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstudio/connect-client/internal/accounts"
	"github.com/rstudio/connect-client/internal/bundles"
	"github.com/rstudio/connect-client/internal/util/utiltest"
	"github.com/rstudio/platform-lib/pkg/rslog"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type DeploymentSuite struct {
	utiltest.Suite
}

func TestDeploymentSuite(t *testing.T) {
	suite.Run(t, new(DeploymentSuite))
}

func (s *DeploymentSuite) TestNewDeployment() {
	d := NewDeployment()
	s.Equal(1, d.Manifest.Version)
}

func (s *DeploymentSuite) TestMergeEmpty() {
	orig := NewDeployment()
	orig.SourceDir = "/my/dir"
	orig.PythonRequirements = []byte("numpy\npandas\n")

	merged := orig
	other := NewDeployment()
	merged.Merge(other)
	s.Equal(orig, merged)
}

func (s *DeploymentSuite) TestMergeNonEmpty() {
	orig := NewDeployment()
	orig.SourceDir = "/my/dir"
	orig.PythonRequirements = []byte("numpy\npandas\n")

	merged := orig
	other := NewDeployment()
	other.SourceDir = "/other/dir"
	other.PythonRequirements = []byte("flask\n")
	merged.Merge(other)
	s.Equal(merged.SourceDir, "/other/dir")
	s.Equal(merged.PythonRequirements, []byte("numpy\npandas\nflask\n"))
}

func (s *DeploymentSuite) TestLoadManifest() {
	manifestJson := []byte(`{"version": 1, "platform": "4.1.0"}`)
	filename := "manifest.json"

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, filename, manifestJson, 0600)
	s.Nil(err)

	deployment := NewDeployment()
	logger := rslog.NewDiscardingLogger()
	err = deployment.LoadManifest(fs, filename, logger)
	s.Nil(err)
	s.Equal(bundles.Manifest{
		Version:  1,
		Platform: "4.1.0",
		Packages: bundles.PackageMap{},
		Files:    bundles.FileMap{},
	}, deployment.Manifest)
}

func (s *DeploymentSuite) TestLoadManifestDir() {
	manifestJson := []byte(`{"version": 1, "platform": "4.1.0"}`)
	filename := "manifest.json"

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, filename, manifestJson, 0600)
	s.Nil(err)

	deployment := NewDeployment()
	logger := rslog.NewDiscardingLogger()
	err = deployment.LoadManifest(fs, filepath.Dir(filename), logger)
	s.Nil(err)
	s.Equal(bundles.Manifest{
		Version:  1,
		Platform: "4.1.0",
		Packages: bundles.PackageMap{},
		Files:    bundles.FileMap{},
	}, deployment.Manifest)
}

func (s *DeploymentSuite) TestLoadManifestNonexistentDir() {
	fs := afero.NewMemMapFs()
	deployment := NewDeployment()
	logger := rslog.NewDiscardingLogger()
	err := deployment.LoadManifest(fs, "/nonexistent", logger)
	s.NotNil(err)
	s.ErrorIs(err, os.ErrNotExist)
}

func (s *DeploymentSuite) TestLoadManifestNonexistentFile() {
	fs := afero.NewMemMapFs()
	dir := "/my/dir"
	fs.MkdirAll(dir, 0700)
	deployment := NewDeployment()
	logger := rslog.NewDiscardingLogger()
	err := deployment.LoadManifest(fs, dir, logger)
	s.NotNil(err)
	s.ErrorIs(err, os.ErrNotExist)
}

func (s *DeploymentSuite) TestSaveLoad() {
	fs := afero.NewMemMapFs()
	dir := "/my/dir"
	logger := rslog.NewDiscardingLogger()
	deployment := NewDeployment()
	deployment.Target.ServerType = accounts.ServerTypeConnect

	configName := "staging"
	err := deployment.SaveToFiles(fs, dir, configName, logger)
	s.Nil(err)
	loadedData := *deployment
	err = deployment.LoadFromFiles(fs, dir, configName, logger)
	s.Nil(err)
	s.Equal(deployment, &loadedData)
}

func (s *DeploymentSuite) TestSaveToFilesErr() {
	fs := utiltest.NewMockFs()
	testError := errors.New("test error from MkdirAll")
	fs.On("MkdirAll", mock.Anything, mock.Anything).Return(testError)
	logger := rslog.NewDiscardingLogger()
	deployment := NewDeployment()
	err := deployment.SaveToFiles(fs, "/nonexistent", "staging", logger)
	s.NotNil(err)
	s.ErrorIs(err, testError)
	fs.AssertExpectations(s.T())
}

func (s *DeploymentSuite) TestSaveErr() {
	serializer := NewMockSerializer()
	testError := errors.New("test error from Save")
	serializer.On("Save", mock.Anything, mock.Anything).Return(testError)
	deployment := NewDeployment()
	err := deployment.Save(serializer)
	s.NotNil(err)
	s.ErrorIs(err, testError)
	serializer.AssertExpectations(s.T())
}

func (s *DeploymentSuite) TestSaveConnectErr() {
	serializer := NewMockSerializer()
	testError := errors.New("test error from Save")
	serializer.On("Save", idLabel, mock.Anything).Return(nil)
	serializer.On("Save", mock.Anything, mock.Anything).Return(testError)
	deployment := NewDeployment()
	deployment.Target.ServerType = accounts.ServerTypeConnect
	err := deployment.Save(serializer)
	s.NotNil(err)
	s.ErrorIs(err, testError)
	serializer.AssertExpectations(s.T())
}

func (s *DeploymentSuite) TestLoadErr() {
	serializer := NewMockSerializer()
	testError := errors.New("test error from Load")
	serializer.On("Load", mock.Anything, mock.Anything).Return(testError)
	deployment := NewDeployment()
	err := deployment.Load(serializer)
	s.NotNil(err)
	s.ErrorIs(err, testError)
	serializer.AssertExpectations(s.T())
}

func (s *DeploymentSuite) TestLoadConnectErr() {
	serializer := NewMockSerializer()
	testError := errors.New("test error from Load")
	serializer.On("Load", idLabel, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		target := args.Get(1).(*TargetID)
		target.ServerType = accounts.ServerTypeConnect
	})
	serializer.On("Load", mock.Anything, mock.Anything).Return(testError)
	deployment := NewDeployment()
	err := deployment.Load(serializer)
	s.NotNil(err)
	s.ErrorIs(err, testError)
	serializer.AssertExpectations(s.T())
}
