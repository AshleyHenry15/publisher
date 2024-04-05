package publish

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"fmt"
	"io"
	"maps"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/rstudio/connect-client/internal/accounts"
	"github.com/rstudio/connect-client/internal/bundles"
	"github.com/rstudio/connect-client/internal/clients/connect"
	"github.com/rstudio/connect-client/internal/deployment"
	"github.com/rstudio/connect-client/internal/events"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/project"
	"github.com/rstudio/connect-client/internal/schema"
	"github.com/rstudio/connect-client/internal/state"
	"github.com/rstudio/connect-client/internal/types"
)

type Publisher interface {
	PublishDirectory(logging.Logger) error
}

type defaultPublisher struct {
	*state.State
	emitter events.Emitter
}

type baseEventData struct {
	LocalID state.LocalDeploymentID `mapstructure:"localId"`
}

type publishStartData struct {
	Server string `mapstructure:"server"`
}

type publishSuccessData struct {
	ContentID    types.ContentID `mapstructure:"contentId"`
	DashboardURL string          `mapstructure:"dashboardUrl"`
	DirectURL    string          `mapstructure:"directUrl"`
	ServerURL    string          `mapstructure:"serverUrl"`
}

type publishFailureData struct {
	Message string `mapstructure:"message"`
}

type publishDeployedFailureData struct {
	DashboardURL string `mapstructure:"dashboardUrl"`
	DirectURL    string `mapstructure:"url"`
}

func NewFromState(s *state.State, emitter events.Emitter) (Publisher, error) {
	if s.LocalID != "" {
		data := baseEventData{
			LocalID: s.LocalID,
		}
		var dataMap events.EventData
		err := mapstructure.Decode(data, &dataMap)
		if err != nil {
			return nil, err
		}
		emitter = events.NewDataEmitter(dataMap, emitter)
	}
	return &defaultPublisher{
		State:   s,
		emitter: emitter,
	}, nil
}

func getDashboardURL(accountURL string, contentID types.ContentID) string {
	return fmt.Sprintf("%s/connect/#/apps/%s", accountURL, contentID)
}

func getDirectURL(accountURL string, contentID types.ContentID) string {
	return fmt.Sprintf("%s/content/%s", accountURL, contentID)
}

func getBundleURL(accountURL string, contentID types.ContentID, bundleID types.BundleID) string {
	return fmt.Sprintf("%s/__api__/v1/content/%s/bundles/%s/download", accountURL, contentID, bundleID)
}

func logAppInfo(w io.Writer, accountURL string, contentID types.ContentID, log logging.Logger, publishingErr error) {
	dashboardURL := getDashboardURL(accountURL, contentID)
	directURL := getDirectURL(accountURL, contentID)
	if publishingErr != nil {
		if contentID == "" {
			// Publishing failed before a content ID was known
			return
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Dashboard URL: ", dashboardURL)
	} else {
		log.Info("Deployment information",
			logging.LogKeyOp, events.AgentOp,
			"dashboardURL", dashboardURL,
			"directURL", directURL,
			"serverURL", accountURL,
			"contentID", contentID,
		)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Dashboard URL: ", dashboardURL)
		fmt.Fprintln(w, "Direct URL:    ", directURL)
	}
}

func (p *defaultPublisher) PublishDirectory(log logging.Logger) error {
	log.Info("Publishing from directory", logging.LogKeyOp, events.AgentOp, "path", p.Dir)
	manifest := bundles.NewManifestFromConfig(p.Config)
	bundler, err := bundles.NewBundler(p.Dir, manifest, nil, log)
	if err != nil {
		return err
	}
	return p.publish(bundler, log)
}

func (p *defaultPublisher) isDeployed() bool {
	return p.Target != nil && p.Target.ID != ""
}

func (p *defaultPublisher) emitErrorEvents(err error, log logging.Logger) {
	agentErr := types.AsAgentError(err)
	dashboardURL := ""
	directURL := ""

	var data events.EventData

	mapstructure.Decode(publishFailureData{
		Message: agentErr.Error(),
	}, &data)

	// Record the error in the deployment record
	if p.Target != nil {
		p.Target.Error = agentErr
		writeErr := p.writeDeploymentRecord(log)
		if writeErr != nil {
			log.Warn("failed to write updated deployment record", "name", p.TargetName, "err", writeErr)
		}
		if p.isDeployed() {
			// Provide URL in the event, if we got far enough in the deployment.
			dashboardURL = getDashboardURL(p.Account.URL, p.Target.ID)
			directURL = getDirectURL(p.Account.URL, p.Target.ID)

			mapstructure.Decode(publishDeployedFailureData{
				DashboardURL: dashboardURL,
				DirectURL:    directURL,
			}, &data)
		}
	}

	maps.Copy(data, agentErr.GetData())

	// Fail the phase
	p.emitter.Emit(events.New(
		agentErr.GetOperation(),
		events.FailurePhase,
		agentErr.GetCode(),
		data))

	// Then fail the publishing operation as a whole
	p.emitter.Emit(events.New(
		events.PublishOp,
		events.FailurePhase,
		agentErr.GetCode(),
		data))
}

func (p *defaultPublisher) publish(
	bundler bundles.Bundler,
	log logging.Logger) error {

	p.emitter.Emit(events.New(events.PublishOp, events.StartPhase, types.NoError, publishStartData{
		Server: p.Account.URL,
	}))
	log.Info("Starting deployment to server", "server", p.Account.URL)

	// TODO: factory method to create client based on server type
	// TODO: timeout option
	client, err := connect.NewConnectClient(p.Account, 2*time.Minute, p.emitter, log)
	if err != nil {
		return err
	}
	err = p.publishWithClient(bundler, p.Account, client, log)
	if p.isDeployed() {
		logAppInfo(os.Stderr, p.Account.URL, p.Target.ID, log, err)
	}
	if err != nil {
		p.emitErrorEvents(err, log)
	} else {
		p.emitter.Emit(events.New(events.PublishOp, events.SuccessPhase, types.NoError, publishSuccessData{
			DashboardURL: getDashboardURL(p.Account.URL, p.Target.ID),
			DirectURL:    getDirectURL(p.Account.URL, p.Target.ID),
			ServerURL:    p.Account.URL,
			ContentID:    p.Target.ID,
		}))
	}
	return err
}

func (p *defaultPublisher) writeDeploymentRecord(log logging.Logger) error {
	if p.SaveName == "" {
		// Redeployment
		p.SaveName = p.TargetName
	} else {
		// Initial deployment
		p.TargetName = p.SaveName
	}

	now := time.Now().Format(time.RFC3339)
	p.Target.DeployedAt = now

	recordPath := deployment.GetDeploymentPath(p.Dir, p.SaveName)
	return p.Target.WriteFile(recordPath)
}

func (p *defaultPublisher) createDeploymentRecord(
	contentID types.ContentID,
	account *accounts.Account,
	log logging.Logger) error {

	// Initial deployment record doesn't know the files or
	// bundleID. These will be added after the
	// bundle upload.
	cfg := *p.Config

	created := ""

	if p.Target != nil {
		created = p.Target.CreatedAt
	} else {
		created = time.Now().Format(time.RFC3339)
	}

	p.Target = &deployment.Deployment{
		Schema:        schema.DeploymentSchemaURL,
		ServerType:    account.ServerType,
		ServerURL:     account.URL,
		ClientVersion: project.Version,
		CreatedAt:     created,
		ID:            contentID,
		ConfigName:    p.ConfigName,
		Files:         nil,
		Configuration: &cfg,
		BundleID:      "",
		DashboardURL:  getDashboardURL(p.Account.URL, contentID),
		DirectURL:     getDirectURL(p.Account.URL, contentID),
		Error:         nil,
	}

	// Save current deployment information for this target
	return p.writeDeploymentRecord(log)
}

func (p *defaultPublisher) publishWithClient(
	bundler bundles.Bundler,
	account *accounts.Account,
	client connect.APIClient,
	log logging.Logger) error {

	var err error

	agentErr := p.checkConfiguration(client, log)
	if agentErr != nil {
		return agentErr
	}

	var contentID types.ContentID
	if p.isDeployed() {
		contentID = p.Target.ID
		log.Info("Updating deployment", "content_id", contentID)
	} else {
		// Create a new deployment; we will update it with details later.
		contentID, err = p.createDeployment(client, log)
		if err != nil {
			return err
		}
	}
	err = p.createDeploymentRecord(contentID, account, log)
	if err != nil {
		return types.AsAgentErrForOperation(events.PublishCreateNewDeploymentOp, err)
	}

	bundleID, err := p.createAndUploadBundle(client, bundler, contentID, log)
	if err != nil {
		return err
	}

	err = p.updateContent(client, contentID, log)
	if err != nil {
		return err
	}

	err = p.setEnvVars(client, contentID, log)
	if err != nil {
		return err
	}

	taskID, err := p.deployBundle(client, contentID, bundleID, log)
	if err != nil {
		return err
	}

	taskLogger := log.WithArgs("source", "server.log")
	err = client.WaitForTask(taskID, taskLogger)
	if err != nil {
		return err
	}

	if p.Config.Validate {
		err = p.validateContent(client, contentID, log)
		if err != nil {
			return err
		}
	}
	return nil
}
