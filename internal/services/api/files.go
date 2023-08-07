package api

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"time"

	"github.com/rstudio/connect-client/internal/bundles/gitignore"
	"github.com/rstudio/connect-client/internal/services/api/pathnames"
	"github.com/rstudio/connect-client/internal/util"
	"github.com/rstudio/platform-lib/pkg/rslog"
	"github.com/spf13/afero"
)

type file struct {
	FileType         fileType `json:"file_type"`         // the file type
	Pathname         string   `json:"pathname"`          // the pathname
	Size             int64    `json:"size"`              // nullable; length in bytes for regular files; system-dependent
	ModifiedDatetime string   `json:"modified_datetime"` // the last modified datetime
	IsDir            bool     `json:"is_dir"`            // true if the file is a directory
	IsEntrypoint     bool     `json:"is_entrypoint"`     // true if the file is an entrypoint
	IsRegular        bool     `json:"is_file"`           // true if the file is a regular file
	IsExcluded       bool     `json:"is_excluded"`       // true if the file is excluded
	Files            []*file  `json:"files"`             // an array of objects of the same type for each file within the directory.
}

func newFile(path util.Path, isExcluded bool) (*file, error) {
	info, err := path.Stat()
	if err != nil {
		return nil, err
	}

	filetype, err := getFileType(path.Path(), info)
	if err != nil {
		return nil, err
	}

	return &file{
		FileType:         filetype,
		Pathname:         path.Path(),
		Size:             info.Size(),
		ModifiedDatetime: info.ModTime().Format(time.RFC3339),
		IsDir:            info.Mode().IsDir(),
		IsRegular:        info.Mode().IsRegular(),
		IsExcluded:       isExcluded,
		Files:            make([]*file, 0),
	}, nil
}

func (f *file) insert(path util.Path, ignore gitignore.GitIgnoreList) (*file, error) {

	if f.Pathname == path.Path() {
		return f, nil
	}

	directory := path.Dir()
	if f.Pathname == directory.Path() {
		for _, child := range f.Files {
			if child.Pathname == path.Path() {
				return child, nil
			}
		}

		isExcluded := ignore.Match(path.Path())
		child, err := newFile(path, isExcluded)
		if err != nil {
			return nil, err
		}

		f.Files = append(f.Files, child)
		return child, nil
	}

	parent, err := f.insert(directory, ignore)
	if err != nil {
		return nil, err
	}

	return parent.insert(path, ignore)
}

func NewFilesController(fs afero.Fs, log rslog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getFile(fs, log, w, r)
		default:
			return
		}
	}
}

func getFile(afs afero.Fs, log rslog.Logger, w http.ResponseWriter, r *http.Request) {
	var p pathnames.Pathname
	if q := r.URL.Query(); q.Has("pathname") {
		p = pathnames.Create(q.Get("pathname"), afs )
	} else {
		p = pathnames.Create(".", afs)
	}

	ok, err := p.IsSafe(log)
	if err != nil {
		internalError(w, log, err)
		return
	}

	// if pathname is not safe, return 403 - Forbidden
	if !ok {
		log.Warnf("the pathname '%s' is not safe", p)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(http.StatusText(http.StatusForbidden)))
		return
	}

	path := util.NewPath(p.String(), afs)
	file, err := toFile(path, log)
	if err != nil {
		internalError(w, log, err)
		return
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(file)
}

func toFile(path util.Path, log rslog.Logger) (*file, error) {
	path = path.Clean()
	ignore := gitignore.New(path)
	isExcluded := ignore.Match(path.Path())
	root, err := newFile(path, isExcluded)
	if err != nil {
		return nil, err
	}

	walker := util.NewSymlinkWalker(util.FSWalker{}, log)
	walker.Walk(path, func(path util.Path, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		_, err = root.insert(path, ignore)
		return err
	})

	return root, nil
}
