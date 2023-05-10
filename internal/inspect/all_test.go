package inspect

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"os"
	"testing"

	"github.com/rstudio/connect-client/internal/apptypes"
	"github.com/rstudio/connect-client/internal/util"
	"github.com/rstudio/connect-client/internal/util/utiltest"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type AllSuite struct {
	utiltest.Suite
}

func TestAllSuite(t *testing.T) {
	suite.Run(t, new(AllSuite))
}

func (s *AllSuite) TestInferTypeDirectory() {
	path := util.NewPath(".", afero.NewMemMapFs())
	htmlFilename := "myfile.html"
	err := path.Join(htmlFilename).WriteFile([]byte("<html></html>\n"), 0600)
	s.Nil(err)

	appFilename := "myapp.py"
	err = path.Join(appFilename).WriteFile([]byte("import dash\n"), 0600)
	s.Nil(err)

	detector := NewContentTypeDetector()
	t, err := detector.InferType(path)
	s.Nil(err)
	s.Equal(&ContentType{
		AppMode:        apptypes.PythonDashMode,
		Entrypoint:     appFilename,
		RequiresPython: true,
	}, t)
}

func (s *AllSuite) TestInferTypeFileLowerPriority() {
	path := util.NewPath(".", afero.NewMemMapFs())

	htmlFilename := "myfile.html"
	htmlPath := path.Join(htmlFilename)
	err := htmlPath.WriteFile([]byte("<html></html>\n"), 0600)
	s.Nil(err)

	appFilename := "myapp.py"
	err = path.Join(appFilename).WriteFile([]byte("import dash\n"), 0600)
	s.Nil(err)

	detector := NewContentTypeDetector()
	t, err := detector.InferType(htmlPath)
	s.Nil(err)
	s.Equal(&ContentType{
		AppMode:    apptypes.StaticMode,
		Entrypoint: htmlFilename,
	}, t)
}

func (s *AllSuite) TestInferTypeFileHigherPriority() {
	path := util.NewPath(".", afero.NewMemMapFs())

	htmlFilename := "myfile.html"
	err := path.Join(htmlFilename).WriteFile([]byte("<html></html>\n"), 0600)
	s.Nil(err)

	appFilename := "myapp.py"
	appPath := path.Join(appFilename)
	err = appPath.WriteFile([]byte("import dash\n"), 0600)
	s.Nil(err)

	detector := NewContentTypeDetector()
	t, err := detector.InferType(appPath)
	s.Nil(err)
	s.Equal(&ContentType{
		AppMode:        apptypes.PythonDashMode,
		Entrypoint:     appFilename,
		RequiresPython: true,
	}, t)
}

func (s *AllSuite) TestInferTypeDirectoryPriority() {
	path := util.NewPath(".", afero.NewMemMapFs())

	htmlFilename := "myfile.html"
	err := path.Join(htmlFilename).WriteFile([]byte("<html></html>\n"), 0600)
	s.Nil(err)

	appFilename := "myapp.py"
	err = path.Join(appFilename).WriteFile([]byte("import dash\n"), 0600)
	s.Nil(err)

	detector := NewContentTypeDetector()
	t, err := detector.InferType(path)
	s.Nil(err)
	s.Equal(&ContentType{
		AppMode:        apptypes.PythonDashMode,
		Entrypoint:     appFilename,
		RequiresPython: true,
	}, t)
}

func (s *AllSuite) TestInferTypeDirectoryIndeterminate() {
	path := util.NewPath(".", afero.NewMemMapFs())
	err := path.Join("myfile").WriteFile([]byte("This is a text file, silly!\n"), 0600)
	s.Nil(err)

	detector := NewContentTypeDetector()
	t, err := detector.InferType(path)
	s.NotNil(err)
	s.ErrorIs(err, errCantDetectContentType)
	s.Nil(t)
}

func (s *AllSuite) TestInferTypeFileIndeterminate() {
	path := util.NewPath("myfile", afero.NewMemMapFs())
	err := path.WriteFile([]byte("This is a text file, silly!\n"), 0600)
	s.Nil(err)

	detector := NewContentTypeDetector()
	t, err := detector.InferType(path)
	s.NotNil(err)
	s.ErrorIs(err, errCantDetectContentType)
	s.Nil(t)
}

func (s *AllSuite) TestInferTypeErr() {
	fs := afero.NewMemMapFs()
	detector := NewContentTypeDetector()
	path := util.NewPath("/foo", fs)
	t, err := detector.InferType(path)
	s.NotNil(err)
	s.ErrorIs(err, os.ErrNotExist)
	s.Nil(t)
}
