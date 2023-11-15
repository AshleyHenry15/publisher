package publish

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rstudio/connect-client/internal/accounts"
	"github.com/rstudio/connect-client/internal/api_client/clients"
	"github.com/rstudio/connect-client/internal/bundles"
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/events"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/state"
	"github.com/rstudio/connect-client/internal/types"
	"github.com/rstudio/connect-client/internal/util"
)

type Publisher struct {
	dir     util.Path
	account *accounts.Account
	cfg     *config.Config
	target  *config.Deployment
}

func New(
	dir util.Path,
	account *accounts.Account,
	cfg *config.Config,
	target *config.Deployment) *Publisher {

	return &Publisher{
		dir:     dir,
		account: account,
		cfg:     cfg,
		target:  target,
	}
}

func (p *Publisher) CreateBundleFromDirectory(dest util.Path, log logging.Logger) error {
	bundleFile, err := dest.Create()
	if err != nil {
		return err
	}
	defer bundleFile.Close()
	bundler, err := bundles.NewBundler(p.dir, bundles.NewManifest(), nil, log)
	if err != nil {
		return err
	}
	_, err = bundler.CreateBundle(bundleFile)
	return err
}

type appInfo struct {
	DashboardURL string `json:"dashboard-url"`
	DirectURL    string `json:"direct-url"`
}

func (p *Publisher) logAppInfo(accountURL string, contentID types.ContentID, log logging.Logger) error {
	appInfo := appInfo{
		DashboardURL: fmt.Sprintf("%s/connect/#/apps/%s", accountURL, contentID),
		DirectURL:    fmt.Sprintf("%s/content/%s", accountURL, contentID),
	}
	log.Success("Deployment successful",
		logging.LogKeyOp, events.PublishOp,
		"dashboardURL", appInfo.DashboardURL,
		"directURL", appInfo.DirectURL,
		"serverURL", accountURL,
		"contentID", contentID,
	)
	jsonInfo, err := json.Marshal(appInfo)
	if err != nil {
		return err
	}
	_, err = fmt.Printf("%s\n", jsonInfo)
	return err
}

func (p *Publisher) PublishDirectory(log logging.Logger) error {
	log.Info("Publishing from directory", "path", p.dir)
	bundler, err := bundles.NewBundler(p.dir, bundles.NewManifest(), nil, log)
	if err != nil {
		return err
	}
	return p.publish(bundler, log)
}

func (p *Publisher) publish(
	bundler bundles.Bundler,
	log logging.Logger) error {

	// TODO: factory method to create client based on server type
	// TODO: timeout option
	client, err := clients.NewConnectClient(p.account, 2*time.Minute, log)
	if err != nil {
		return err
	}
	err = p.publishWithClient(bundler, p.account, client, log)
	if err != nil {
		log.Failure(err)
	}
	return nil
}

type DeploymentNotFoundDetails struct {
	ContentID types.ContentID
}

func withLog[T any](
	op events.Operation,
	msg string,
	label string,
	log logging.Logger,
	fn func() (T, error)) (value T, err error) {

	log = log.WithArgs(logging.LogKeyOp, op)
	log.Start(msg)
	value, err = fn()
	if err != nil {
		// Using implicit return values here since we
		// can't construct an empty T{}
		err = types.ErrToAgentError(op, err)
		return
	}
	log.Success("Done", label, value)
	return value, nil
}

func (p *Publisher) publishWithClient(
	bundler bundles.Bundler,
	account *accounts.Account,
	client clients.APIClient,
	log logging.Logger) error {

	log.Start("Starting deployment to server",
		logging.LogKeyOp, events.PublishOp,
		"server", account.URL,
	)

	bundleFile, err := os.CreateTemp("", "bundle-*.tar.gz")
	if err != nil {
		return types.ErrToAgentError(events.PublishCreateBundleOp, err)
	}
	defer os.Remove(bundleFile.Name())
	defer bundleFile.Close()

	_, err = withLog(events.PublishCreateBundleOp, "Creating bundle", "filename", log, func() (any, error) {
		_, err := bundler.CreateBundle(bundleFile)
		return bundleFile.Name(), err
	})
	if err != nil {
		return types.ErrToAgentError(events.PublishCreateBundleOp, err)
	}
	_, err = bundleFile.Seek(0, io.SeekStart)
	if err != nil {
		return types.ErrToAgentError(events.PublishCreateBundleOp, err)
	}

	var contentID types.ContentID
	connectContent := state.ConnectContentFromConfig(p.cfg)
	if p.target != nil {
		contentID = p.target.Id
		_, err := withLog(events.PublishCreateDeploymentOp, "Updating deployment", "content_id", log, func() (any, error) {
			return contentID, client.UpdateDeployment(contentID, connectContent)
		})
		if err != nil {
			httpErr, ok := err.(*clients.HTTPError)
			if ok && httpErr.Status == http.StatusNotFound {
				details := DeploymentNotFoundDetails{
					ContentID: contentID,
				}
				return types.NewAgentError(events.DeploymentNotFoundCode, err, details)
			}
			return err
		}
	} else {
		contentID, err = withLog(events.PublishCreateDeploymentOp, "Creating deployment", "content_id", log, func() (types.ContentID, error) {
			return client.CreateDeployment(connectContent)
		})
		if err != nil {
			return err
		}
		log.Info("content_id", contentID)
	}

	bundleID, err := withLog(events.PublishUploadBundleOp, "Uploading deployment bundle", "bundle_id", log, func() (types.BundleID, error) {
		return client.UploadBundle(contentID, bundleFile)
	})
	if err != nil {
		return err
	}

	p.target = &config.Deployment{
		ServerType: account.ServerType,
		ServerURL:  account.URL,
		Id:         contentID,
	}

	taskID, err := withLog(events.PublishDeployBundleOp, "Initiating bundle deployment", "task_id", log, func() (types.TaskID, error) {
		return client.DeployBundle(contentID, bundleID)
	})
	if err != nil {
		return err
	}

	taskLogger := log.WithArgs("source", "serverp.log")
	err = client.WaitForTask(taskID, taskLogger)
	if err != nil {
		return err
	}
	log = log.WithArgs(logging.LogKeyOp, events.AgentOp)
	return p.logAppInfo(account.URL, contentID, log)
}
