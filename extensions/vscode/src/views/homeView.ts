// Copyright (C) 2024 by Posit Software, PBC.

import {
  CancellationToken,
  Disposable,
  ExtensionContext,
  ThemeIcon,
  Uri,
  Webview,
  WebviewView,
  WebviewViewProvider,
  WebviewViewResolveContext,
  WorkspaceFolder,
  commands,
  env,
  window,
  workspace,
} from "vscode";
import { isAxiosError } from "axios";

import {
  Configuration,
  ConfigurationError,
  Credential,
  Deployment,
  EventStreamMessage,
  FileAction,
  PreDeployment,
  PreDeploymentWithConfig,
  isConfigurationError,
  isDeploymentError,
  isPreDeployment,
  isPreDeploymentWithConfig,
  useApi,
} from "src/api";
import { useBus } from "src/bus";
import { EventStream } from "src/events";
import { getPythonInterpreterPath } from "../utils/config";
import { getSummaryStringFromError } from "src/utils/errors";
import { getNonce } from "src/utils/getNonce";
import { getUri } from "src/utils/getUri";
import { deployProject } from "src/views/deployProgress";
import { WebviewConduit } from "src/utils/webviewConduit";
import { fileExists } from "src/utils/files";
import { newDestination } from "src/multiStepInputs/newDestination";

import type { DestinationNames, HomeViewState } from "src/types/shared";
import {
  DeployMsg,
  EditConfigurationMsg,
  NavigateMsg,
  SaveSelectionStatedMsg,
  WebviewToHostMessage,
  WebviewToHostMessageType,
  VSCodeOpenRelativeMsg,
} from "src/types/messages/webviewToHostMessages";
import { HostToWebviewMessageType } from "src/types/messages/hostToWebviewMessages";
import { confirmOverwrite } from "src/dialogs";
import { splitFilesOnInclusion } from "src/utils/files";
import { DestinationQuickPick } from "src/types/quickPicks";
import { normalizeURL } from "src/utils/url";
import { selectConfig } from "src/multiStepInputs/selectConfig";
import { RPackage, RVersionConfig } from "src/api/types/packages";
import { ConfigWatcherManager, WatcherManager } from "src/watchers";

const viewName = "posit.publisher.homeView";
const refreshCommand = viewName + ".refresh";
const selectConfigForDestination = viewName + ".selectConfigForDestination";
const selectDestinationCommand = viewName + ".selectDestination";
const newDestinationCommand = viewName + ".newDestination";
const contextIsHomeViewInitialized = viewName + ".initialized";

enum HomeViewInitialized {
  initialized = "initialized",
  uninitialized = "uninitialized",
}

const lastSelectionState = viewName + ".lastSelectionState.v2";

export class HomeViewProvider implements WebviewViewProvider, Disposable {
  private _disposables: Disposable[] = [];
  private _deployments: (
    | Deployment
    | PreDeployment
    | PreDeploymentWithConfig
  )[] = [];

  private _credentials: Credential[] = [];
  private _configs: Configuration[] = [];
  private configsInError: ConfigurationError[] = [];
  private root: WorkspaceFolder | undefined;
  private _webviewView?: WebviewView;
  private _extensionUri: Uri;
  private _webviewConduit: WebviewConduit;

  private configWatchers: ConfigWatcherManager | undefined;

  constructor(
    private readonly _context: ExtensionContext,
    private readonly _stream: EventStream,
  ) {
    const workspaceFolders = workspace.workspaceFolders;
    if (workspaceFolders !== undefined) {
      this.root = workspaceFolders[0];
    }
    this._extensionUri = this._context.extensionUri;
    this._webviewConduit = new WebviewConduit();

    // if someone needs a refresh of any active params,
    // we are here to service that request!
    useBus().on("refreshCredentials", async () => {
      await this._refreshCredentialData();
      this._updateWebViewViewCredentials();
    });
    useBus().on("requestActiveConfig", () => {
      useBus().trigger("activeConfigChanged", this._getActiveConfig());
    });
    useBus().on("requestActiveDeployment", () => {
      useBus().trigger("activeDeploymentChanged", this._getActiveDeployment());
    });

    useBus().on("activeConfigChanged", (cfg) => {
      this.sendRefreshedFilesLists();
      this._onRefreshPythonPackages();
      this._onRefreshRPackages();

      this.configWatchers?.dispose();
      this.configWatchers = new ConfigWatcherManager(cfg);

      this.configWatchers.configFile?.onDidChange(
        this.sendRefreshedFilesLists,
        this,
      );

      this.configWatchers.pythonPackageFile?.onDidCreate(
        this._onRefreshPythonPackages,
        this,
      );
      this.configWatchers.pythonPackageFile?.onDidChange(
        this._onRefreshPythonPackages,
        this,
      );
      this.configWatchers.pythonPackageFile?.onDidDelete(
        this._onRefreshPythonPackages,
        this,
      );

      this.configWatchers.rPackageFile?.onDidCreate(
        this._onRefreshRPackages,
        this,
      );
      this.configWatchers.rPackageFile?.onDidChange(
        this._onRefreshRPackages,
        this,
      );
      this.configWatchers.rPackageFile?.onDidDelete(
        this._onRefreshRPackages,
        this,
      );
    });
  }
  /**
   * Dispatch messages passed from the webview to the handling code
   */
  private async _onConduitMessage(msg: WebviewToHostMessage) {
    switch (msg.kind) {
      case WebviewToHostMessageType.DEPLOY:
        return await this._onDeployMsg(msg);
      case WebviewToHostMessageType.INITIALIZING:
        return await this._onInitializingMsg();
      case WebviewToHostMessageType.NEW_DEPLOYMENT:
        return await this._onNewDeploymentMsg();
      case WebviewToHostMessageType.EDIT_CONFIGURATION:
        return await this._onEditConfigurationMsg(msg);
      case WebviewToHostMessageType.NEW_CONFIGURATION:
        return await this._onNewConfigurationMsg();
      case WebviewToHostMessageType.SELECT_CONFIGURATION:
        return await this.selectConfigForDestination();
      case WebviewToHostMessageType.NAVIGATE:
        return await this._onNavigateMsg(msg);
      case WebviewToHostMessageType.SAVE_SELECTION_STATE:
        return await this._onSaveSelectionState(msg);
      case WebviewToHostMessageType.REFRESH_PYTHON_PACKAGES:
        return await this._onRefreshPythonPackages();
      case WebviewToHostMessageType.REFRESH_R_PACKAGES:
        return await this._onRefreshRPackages();
      case WebviewToHostMessageType.VSCODE_OPEN_RELATIVE:
        return await this._onRelativeOpenVSCode(msg);
      case WebviewToHostMessageType.SCAN_PYTHON_PACKAGE_REQUIREMENTS:
        return await this._onScanForPythonPackageRequirements();
      case WebviewToHostMessageType.SCAN_R_PACKAGE_REQUIREMENTS:
        return await this._onScanForRPackageRequirements();
      case WebviewToHostMessageType.VSCODE_OPEN:
        return commands.executeCommand(
          "vscode.open",
          Uri.parse(msg.content.uri),
        );
      case WebviewToHostMessageType.REQUEST_FILES_LISTS:
        return this.sendRefreshedFilesLists();
      case WebviewToHostMessageType.INCLUDE_FILE:
        return this.updateFileList(msg.content.path, FileAction.INCLUDE);
      case WebviewToHostMessageType.EXCLUDE_FILE:
        return this.updateFileList(msg.content.path, FileAction.EXCLUDE);
      case WebviewToHostMessageType.SELECT_DESTINATION:
        return this.showDestinationQuickPick();
      case WebviewToHostMessageType.NEW_DESTINATION:
        return this.showNewDestinationMultiStep(viewName);
      case WebviewToHostMessageType.NEW_CREDENTIAL:
        return this.showNewCredential();
      default:
        throw new Error(
          `Error: _onConduitMessage unhandled msg: ${JSON.stringify(msg)}`,
        );
    }
  }

  private async _onDeployMsg(msg: DeployMsg) {
    try {
      const api = await useApi();
      const response = await api.deployments.publish(
        msg.content.deploymentName,
        msg.content.credentialName,
        msg.content.configurationName,
      );
      deployProject(response.data.localId, this._stream);
    } catch (error: unknown) {
      const summary = getSummaryStringFromError("homeView, deploy", error);
      window.showInformationMessage(`Failed to deploy . ${summary}`);
    }
  }

  private async _onInitializingMsg() {
    // send back the data needed.
    await this.refreshAll(true);
    this.setInitializationContext(HomeViewInitialized.initialized);

    // On first run, we have no saved state. Trigger a save
    // so we have the state, and can notify dependent views.
    this._requestWebviewSaveSelection();
  }

  private setInitializationContext(context: HomeViewInitialized) {
    commands.executeCommand(
      "setContext",
      contextIsHomeViewInitialized,
      context,
    );
  }

  private async _onNewDeploymentMsg() {
    const preDeployment: PreDeployment = await commands.executeCommand(
      "posit.publisher.deployments.createNewDeploymentFile",
    );
    if (preDeployment) {
      this._updateDeploymentFileSelection(preDeployment, true);
    }
  }

  private async _onEditConfigurationMsg(msg: EditConfigurationMsg) {
    let config: Configuration | ConfigurationError | undefined;
    config = this._configs.find(
      (config) => config.configurationName === msg.content.configurationName,
    );
    if (!config) {
      config = this.configsInError.find(
        (config) => config.configurationName === msg.content.configurationName,
      );
    }
    if (config) {
      await commands.executeCommand(
        "vscode.open",
        Uri.file(config.configurationPath),
      );
    }
  }

  private async _onNewConfigurationMsg() {
    await commands.executeCommand(
      "posit.publisher.configurations.add",
      viewName,
    );
  }

  private async _onNavigateMsg(msg: NavigateMsg) {
    await env.openExternal(Uri.parse(msg.content.uriPath));
  }

  private async _onSaveSelectionState(msg: SaveSelectionStatedMsg) {
    await this._saveSelectionState(msg.content.state);
  }

  private async updateFileList(uri: string, action: FileAction) {
    const activeConfig = this._getActiveConfig();
    if (activeConfig === undefined) {
      console.error("homeView::updateFileList: No active configuration.");
      return;
    }

    try {
      const api = await useApi();
      await api.files.updateFileList(
        activeConfig.configurationName,
        uri,
        action,
      );
    } catch (error: unknown) {
      const summary = getSummaryStringFromError(
        "homeView::updateFileList",
        error,
      );
      window.showErrorMessage(`Failed to update config file. ${summary}`);
      return;
    }
  }

  private _onPublishStart() {
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.PUBLISH_START,
    });
  }

  private _onPublishSuccess() {
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.PUBLISH_FINISH_SUCCESS,
    });
  }

  private _onPublishFailure(msg: EventStreamMessage) {
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.PUBLISH_FINISH_FAILURE,
      content: {
        data: {
          message: msg.data.message,
        },
      },
    });
  }

  private async _refreshDeploymentData() {
    try {
      // API Returns:
      // 200 - success
      // 500 - internal server error
      const api = await useApi();
      const response = await api.deployments.getAll();
      const deployments = response.data;
      this._deployments = [];
      deployments.forEach((deployment) => {
        if (!isDeploymentError(deployment)) {
          this._deployments.push(deployment);
        }
      });
    } catch (error: unknown) {
      const summary = getSummaryStringFromError(
        "_refreshDeploymentData::deployments.getAll",
        error,
      );
      window.showInformationMessage(summary);
      throw error;
    }
  }

  private async _refreshConfigurationData() {
    try {
      const api = await useApi();
      const response = await api.configurations.getAll();
      const configurations = response.data;
      this._configs = [];
      this.configsInError = [];
      configurations.forEach((config) => {
        if (!isConfigurationError(config)) {
          this._configs.push(config);
        } else {
          this.configsInError.push(config);
        }
      });
    } catch (error: unknown) {
      const summary = getSummaryStringFromError(
        "_refreshConfigurationData::configurations.getAll",
        error,
      );
      window.showInformationMessage(summary);
      throw error;
    }
  }

  private async _refreshCredentialData() {
    try {
      const api = await useApi();
      const response = await api.credentials.list();
      this._credentials = response.data;
    } catch (error: unknown) {
      const summary = getSummaryStringFromError(
        "_refreshCredentialData::credentials.list",
        error,
      );
      window.showInformationMessage(summary);
      throw error;
    }
  }

  private _updateWebViewViewDeployments(
    selectedDeploymentName?: string | null,
  ) {
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.REFRESH_DEPLOYMENT_DATA,
      content: {
        deployments: this._deployments,
        selectedDeploymentName,
      },
    });
  }

  private _updateWebViewViewConfigurations(
    selectedConfigurationName?: string | null,
  ) {
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.REFRESH_CONFIG_DATA,
      content: {
        configurations: this._configs,
        configurationsInError: this.configsInError,
        selectedConfigurationName,
      },
    });
  }

  private _updateWebViewViewCredentials() {
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.REFRESH_CREDENTIAL_DATA,
      content: {
        credentials: this._credentials,
      },
    });
  }

  private _updateDeploymentFileSelection(
    preDeployment: PreDeployment,
    saveSelection = false,
  ) {
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.UPDATE_DEPLOYMENT_SELECTION,
      content: {
        preDeployment,
        saveSelection,
      },
    });
  }

  private _requestWebviewSaveSelection() {
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.SAVE_SELECTION,
    });
  }

  private _getSelectionState(): HomeViewState {
    const state = this._context.workspaceState.get<HomeViewState>(
      lastSelectionState,
      {
        deploymentName: undefined,
        configurationName: undefined,
      },
    );
    return state;
  }

  private _getActiveConfig(): Configuration | undefined {
    const savedState = this._getSelectionState();
    return this.getConfigByName(savedState.configurationName);
  }

  private _getActiveDeployment(): Deployment | PreDeployment | undefined {
    const savedState = this._getSelectionState();
    return this.getDeploymentByName(savedState.deploymentName);
  }

  private getDeploymentByName(name: string | undefined) {
    return this._deployments.find((d) => d.deploymentName === name);
  }

  private getConfigByName(name: string | undefined) {
    return this._configs.find((c) => c.configurationName === name);
  }

  private async _saveSelectionState(state: HomeViewState): Promise<void> {
    await this._context.workspaceState.update(lastSelectionState, state);

    useBus().trigger("activeDeploymentChanged", this._getActiveDeployment());
    useBus().trigger("activeConfigChanged", this._getActiveConfig());
  }

  private async _onRefreshPythonPackages() {
    const savedState = this._getSelectionState();
    const activeConfiguration = savedState.configurationName;
    let pythonProject = true;
    let packages: string[] = [];
    let packageFile: string | undefined;
    let packageMgr: string | undefined;

    const api = await useApi();

    if (activeConfiguration) {
      const currentConfig = this.getConfigByName(activeConfiguration);
      const pythonSection = currentConfig?.configuration.python;
      if (!pythonSection) {
        pythonProject = false;
      } else {
        try {
          packageFile = pythonSection.packageFile;
          packageMgr = pythonSection.packageManager;

          const response =
            await api.packages.getPythonPackages(activeConfiguration);
          packages = response.data.requirements;
        } catch (error: unknown) {
          if (isAxiosError(error) && error.response?.status === 404) {
            // No requirements file or contains invalid entries; show the welcome view.
            packageFile = undefined;
          } else if (isAxiosError(error) && error.response?.status === 422) {
            // invalid package file
            packageFile = undefined;
          } else if (isAxiosError(error) && error.response?.status === 409) {
            // Python is not present in the configuration file
            pythonProject = false;
          } else {
            const summary = getSummaryStringFromError(
              "homeView::_onRefreshPythonPackages",
              error,
            );
            window.showInformationMessage(summary);
            return;
          }
        }
      }
    }
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.UPDATE_PYTHON_PACKAGES,
      content: {
        pythonProject,
        file: packageFile,
        manager: packageMgr,
        packages,
      },
    });
  }

  private async _onRefreshRPackages() {
    const savedState = this._getSelectionState();
    const activeConfiguration = savedState.configurationName;
    let rProject = true;
    let packages: RPackage[] = [];
    let packageFile: string | undefined;
    let packageMgr: string | undefined;
    let rVersionConfig: RVersionConfig | undefined;

    const api = await useApi();

    if (activeConfiguration) {
      const currentConfig = this.getConfigByName(activeConfiguration);
      const rSection = currentConfig?.configuration.r;
      if (!rSection) {
        rProject = false;
      } else {
        try {
          packageFile = rSection.packageFile;
          packageMgr = rSection.packageManager;

          const response = await api.packages.getRPackages(activeConfiguration);
          packages = [];
          Object.keys(response.data.packages).forEach((key: string) =>
            packages.push(response.data.packages[key]),
          );
          rVersionConfig = response.data.r;
        } catch (error: unknown) {
          if (isAxiosError(error) && error.response?.status === 404) {
            // No requirements file; show the welcome view.
            packageFile = undefined;
          } else if (isAxiosError(error) && error.response?.status === 422) {
            // invalid package file
            packageFile = undefined;
          } else if (isAxiosError(error) && error.response?.status === 409) {
            // R is not present in the configuration file
            rProject = false;
          } else {
            const summary = getSummaryStringFromError(
              "homeView::_onRefreshRPackages",
              error,
            );
            window.showInformationMessage(summary);
            return;
          }
        }
      }
    }
    this._webviewConduit.sendMsg({
      kind: HostToWebviewMessageType.UPDATE_R_PACKAGES,
      content: {
        rProject,
        file: packageFile,
        manager: packageMgr,
        rVersion: rVersionConfig?.version,
        packages,
      },
    });
  }

  private async _onRelativeOpenVSCode(msg: VSCodeOpenRelativeMsg) {
    if (this.root === undefined) {
      return;
    }
    const fileUri = Uri.joinPath(this.root.uri, msg.content.relativePath);
    await commands.executeCommand("vscode.open", fileUri);
  }

  private async _onScanForPythonPackageRequirements() {
    if (this.root === undefined) {
      // We shouldn't get here if there's no workspace folder open.
      return;
    }
    const activeConfiguration = this._getActiveConfig();
    const relPathPackageFile =
      activeConfiguration?.configuration.python?.packageFile;
    if (relPathPackageFile === undefined) {
      return;
    }

    const fileUri = Uri.joinPath(this.root.uri, relPathPackageFile);

    if (await fileExists(fileUri)) {
      const ok = await confirmOverwrite(
        `Are you sure you want to overwrite your existing ${relPathPackageFile} file?`,
      );
      if (!ok) {
        return;
      }
    }

    try {
      const api = await useApi();
      const python = await getPythonInterpreterPath();
      await api.packages.createPythonRequirementsFile(
        python,
        relPathPackageFile,
      );
      await commands.executeCommand("vscode.open", fileUri);
    } catch (error: unknown) {
      const summary = getSummaryStringFromError(
        "homeView::_onScanForPythonPackageRequirements",
        error,
      );
      window.showInformationMessage(summary);
    }
  }

  private async _onScanForRPackageRequirements() {
    if (this.root === undefined) {
      // We shouldn't get here if there's no workspace folder open.
      return;
    }
    const activeConfiguration = this._getActiveConfig();
    const relPathPackageFile =
      activeConfiguration?.configuration.r?.packageFile;
    if (relPathPackageFile === undefined) {
      return;
    }

    const fileUri = Uri.joinPath(this.root.uri, relPathPackageFile);

    if (await fileExists(fileUri)) {
      const ok = await confirmOverwrite(
        `Are you sure you want to overwrite your existing ${relPathPackageFile} file?`,
      );
      if (!ok) {
        return;
      }
    }

    try {
      const api = await useApi();
      await api.packages.createRRequirementsFile(relPathPackageFile);
      await commands.executeCommand("vscode.open", fileUri);
    } catch (error: unknown) {
      const summary = getSummaryStringFromError(
        "homeView::_onScanForRPackageRequirements",
        error,
      );
      window.showInformationMessage(summary);
    }
  }

  private async propogateDestinationSelection(
    configurationName?: string,
    deploymentName?: string,
  ) {
    // We have to break our protocol and go ahead and write this into storage,
    // in case this multi-stepper is actually running ahead of the webview
    // being brought up.
    this._saveSelectionState({
      deploymentName,
      configurationName,
    });
    // Now push down into the webview
    this._updateWebViewViewCredentials();
    this._updateWebViewViewConfigurations(configurationName);
    this._updateWebViewViewDeployments(deploymentName);
    // And have the webview save what it has selected.
    this._requestWebviewSaveSelection();
  }

  private async selectConfigForDestination() {
    const config = await selectConfig("Select a Configuration", viewName);
    if (config) {
      const activeDeployment = this._getActiveDeployment();
      if (activeDeployment === undefined) {
        console.error(
          "homeView::selectConfigForDestination: No active deployment.",
        );
        return;
      }
      const api = await useApi();
      await api.deployments.patch(
        activeDeployment.deploymentName,
        config.configurationName,
      );
    }
  }

  public async showNewDestinationMultiStep(
    viewId?: string,
  ): Promise<DestinationNames | undefined> {
    const destinationObjects = await newDestination(viewId);
    if (destinationObjects) {
      // add out new objects into our collections possibly ahead (we don't know) of
      // the file refresh activity (for deployment and config)
      // and the credential refresh that we will kick off
      //
      // Doing this as an alternative to forcing a full refresh
      // of all three APIs prior to updating the UX, which would
      // be seen as a visible delay (we'd have to have a progress indicator).
      let refreshCredentials = false;
      if (
        !this._deployments.find(
          (deployment) =>
            deployment.saveName === destinationObjects.deployment.saveName,
        )
      ) {
        this._deployments.push(destinationObjects.deployment);
      }
      if (
        !this._configs.find(
          (config) =>
            config.configurationName ===
            destinationObjects.configuration.configurationName,
        )
      ) {
        this._configs.push(destinationObjects.configuration);
      }
      if (
        !this._credentials.find(
          (credential) =>
            credential.name === destinationObjects.credential.name,
        )
      ) {
        this._credentials.push(destinationObjects.credential);
        refreshCredentials = true;
      }

      this.propogateDestinationSelection(
        destinationObjects.configuration.configurationName,
        destinationObjects.deployment.saveName,
      );
      // Credentials aren't auto-refreshed, so we have to trigger it ourselves.
      if (refreshCredentials) {
        useBus().trigger("refreshCredentials", undefined);
      }
      return {
        configurationName: destinationObjects.configuration.configurationName,
        deploymentName: destinationObjects.deployment.saveName,
      };
    }
    return undefined;
  }

  private showNewCredential() {
    const deployment = this._getActiveDeployment();

    return commands.executeCommand(
      "posit.publisher.credentials.add",
      deployment?.serverUrl,
    );
  }

  private async showDestinationQuickPick(): Promise<
    DestinationNames | undefined
  > {
    // Create quick pick list from current deployments, credentials and configs
    const destinations: DestinationQuickPick[] = [];
    const lastDeploymentName = this._getActiveDeployment()?.saveName;
    const lastConfigName = this._getActiveConfig()?.configurationName;

    this._deployments.forEach((deployment) => {
      if (
        isDeploymentError(deployment) ||
        (isPreDeployment(deployment) && !isPreDeploymentWithConfig(deployment))
      ) {
        // we won't include these for now. Perhaps in the future, we can show them
        // as disabled.
        return;
      }

      let config: Configuration | undefined;
      if (deployment.configurationName) {
        config = this._configs.find(
          (config) => config.configurationName === deployment.configurationName,
        );
      }

      let credential = this._credentials.find(
        (credential) =>
          normalizeURL(credential.url).toLowerCase() ===
          normalizeURL(deployment.serverUrl).toLowerCase(),
      );

      let title = deployment.saveName;
      let problem = false;

      let configName = config?.configurationName;
      if (!configName) {
        configName = deployment.configurationName
          ? `Missing Configuration ${deployment.configurationName}`
          : `ERROR: No Config Entry in Deployment file - ${deployment.saveName}`;
        problem = true;
      }

      let credentialName = credential?.name;
      if (!credentialName) {
        credentialName = `Missing Credential for ${deployment.serverUrl}`;
        problem = true;
      }

      let lastMatch =
        lastDeploymentName === deployment.saveName &&
        lastConfigName === configName;

      const destination: DestinationQuickPick = {
        label: title,
        description: configName,
        detail: credentialName,
        iconPath: problem
          ? new ThemeIcon("error")
          : new ThemeIcon("cloud-upload"),
        deployment,
        config,
        lastMatch,
      };
      // Should we not push destinations with no config or matching credentials?
      destinations.push(destination);
    });

    const toDispose: Disposable[] = [];
    const destination = await new Promise<DestinationQuickPick | undefined>(
      (resolve) => {
        const quickPick = window.createQuickPick<DestinationQuickPick>();
        this._disposables.push(quickPick);

        quickPick.items = destinations;
        const lastMatches = destinations.filter(
          (destination) => destination.lastMatch,
        );
        if (lastMatches) {
          quickPick.activeItems = lastMatches;
        }
        quickPick.title = "Select Destination";
        quickPick.ignoreFocusOut = true;
        quickPick.matchOnDescription = true;
        quickPick.matchOnDetail = true;
        quickPick.show();

        quickPick.onDidAccept(
          () => {
            quickPick.hide();
            if (quickPick.selectedItems.length > 0) {
              return resolve(quickPick.selectedItems[0]);
            }
            resolve(undefined);
          },
          undefined,
          toDispose,
        );
        quickPick.onDidHide(() => resolve(undefined), undefined, toDispose);
      },
    ).finally(() => Disposable.from(...toDispose).dispose());

    let result: DestinationNames | undefined;
    if (destination) {
      result = {
        deploymentName: destination.deployment.saveName,
        configurationName: destination.deployment.configurationName,
      };
      this._updateWebViewViewCredentials();
      this._updateWebViewViewConfigurations(result.configurationName);
      this._updateWebViewViewDeployments(result.deploymentName);
      this._requestWebviewSaveSelection();
    }
    return result;
  }

  public resolveWebviewView(
    webviewView: WebviewView,
    _: WebviewViewResolveContext,
    _token: CancellationToken,
  ) {
    this._webviewView = webviewView;
    this._webviewConduit.init(this._webviewView.webview);

    // Allow scripts in the webview
    webviewView.webview.options = {
      // Enable JavaScript in the webview
      enableScripts: true,
      // Restrict the webview to only load resources from these directories
      localResourceRoots: [
        Uri.joinPath(this._extensionUri, "webviews", "homeView", "dist"),
        Uri.joinPath(
          this._extensionUri,
          "node_modules",
          "@vscode",
          "codicons",
          "dist",
        ),
      ],
    };

    // Set the HTML content that will fill the webview view
    webviewView.webview.html = this._getWebviewContent(
      webviewView.webview,
      this._extensionUri,
    );

    // Sets up an event listener to listen for messages passed from the webview view this._context
    // and executes code based on the message that is recieved
    this._disposables.push(
      this._webviewConduit.onMsg(this._onConduitMessage.bind(this)),
    );
  }
  /**
   * Defines and returns the HTML that should be rendered within the webview panel.
   *
   * @remarks This is also the place where references to the Vue webview build files
   * are created and inserted into the webview HTML.
   *
   * @param webview A reference to the extension webview
   * @param extensionUri The URI of the directory containing the extension
   * @returns A template string literal containing the HTML that should be
   * rendered within the webview panel
   */
  private _getWebviewContent(webview: Webview, extensionUri: Uri) {
    // The CSS files from the Vue build output
    const stylesUri = getUri(webview, extensionUri, [
      "webviews",
      "homeView",
      "dist",
      "index.css",
    ]);
    // The JS file from the Vue build output
    const scriptUri = getUri(webview, extensionUri, [
      "webviews",
      "homeView",
      "dist",
      "index.js",
    ]);
    // The codicon css (and related tff file) are needing to be loaded for icons
    const codiconsUri = getUri(webview, extensionUri, [
      "node_modules",
      "@vscode",
      "codicons",
      "dist",
      "codicon.css",
    ]);

    const nonce = getNonce();

    // Tip: Install the es6-string-html VS Code extension to enable code highlighting below
    return /*html*/ `
      <!DOCTYPE html>
      <html lang="en">
        <head>
          <meta charset="UTF-8" />
          <meta name="viewport" content="width=device-width, initial-scale=1.0" />
          <meta http-equiv="Content-Security-Policy"
            content="
              default-src 'none';
              font-src ${webview.cspSource};
              style-src ${webview.cspSource} 'unsafe-inline';
              script-src 'nonce-${nonce}';"
          />
          <link rel="stylesheet" type="text/css" href="${stylesUri}">
          <link rel="stylesheet" type="text/css" href="${codiconsUri}">
          <title>Hello World</title>
        </head>
        <body>
          <div id="app"></div>
          <script type="module" nonce="${nonce}" src="${scriptUri}"></script>
        </body>
      </html>
    `;
  }

  public refreshAll = async (includeSavedState?: boolean) => {
    try {
      await Promise.all([
        this._refreshDeploymentData(),
        this._refreshConfigurationData(),
        this._refreshCredentialData(),
      ]);
    } catch (error: unknown) {
      const summary = getSummaryStringFromError(
        "refreshAll::Promise.all",
        error,
      );
      window.showInformationMessage(summary);
      return;
    }
    const selectionState = includeSavedState
      ? this._getSelectionState()
      : undefined;
    this._updateWebViewViewCredentials();
    this._updateWebViewViewConfigurations(
      selectionState?.configurationName || null,
    );
    this._updateWebViewViewDeployments(selectionState?.deploymentName || null);
    if (includeSavedState && selectionState) {
      useBus().trigger("activeDeploymentChanged", this._getActiveDeployment());
      useBus().trigger("activeConfigChanged", this._getActiveConfig());
    }
  };

  public refreshDeployments = async () => {
    await this._refreshDeploymentData();
    this._updateWebViewViewDeployments();
    useBus().trigger("activeDeploymentChanged", this._getActiveDeployment());
  };

  public refreshConfigurations = async () => {
    await this._refreshConfigurationData();
    this._updateWebViewViewConfigurations();
    useBus().trigger("activeConfigChanged", this._getActiveConfig());
  };

  public sendRefreshedFilesLists = async () => {
    const api = await useApi();
    const activeConfig = this._getActiveConfig();
    if (activeConfig) {
      const response = await api.files.getByConfiguration(
        activeConfig.configurationName,
      );

      this._webviewConduit.sendMsg({
        kind: HostToWebviewMessageType.REFRESH_FILES_LISTS,
        content: {
          ...splitFilesOnInclusion(response.data),
        },
      });
    }
  };

  /**
   * Cleans up and disposes of webview resources when view is disposed
   */
  public dispose() {
    Disposable.from(...this._disposables).dispose();

    this.configWatchers?.dispose();
  }

  public register(watchers: WatcherManager) {
    this._stream.register("publish/start", () => {
      this._onPublishStart();
    });
    this._stream.register("publish/success", () => {
      this._onPublishSuccess();
    });
    this._stream.register("publish/failure", (msg: EventStreamMessage) => {
      this._onPublishFailure(msg);
    });

    this._context.subscriptions.push(
      window.registerWebviewViewProvider(viewName, this, {
        webviewOptions: {
          retainContextWhenHidden: true,
        },
      }),
    );

    this._context.subscriptions.push(
      commands.registerCommand(
        selectDestinationCommand,
        this.showDestinationQuickPick,
        this,
      ),
      commands.registerCommand(
        newDestinationCommand,
        () => this.showNewDestinationMultiStep(viewName),
        this,
      ),
    );

    this._context.subscriptions.push(
      commands.registerCommand(refreshCommand, () => this.refreshAll(true)),
      commands.registerCommand(
        selectConfigForDestination,
        this.selectConfigForDestination,
        this,
      ),
    );

    watchers.positDir?.onDidDelete(() => {
      this.refreshDeployments();
      this.refreshConfigurations();
    }, this);
    watchers.publishDir?.onDidDelete(() => {
      this.refreshDeployments();
      this.refreshConfigurations();
    }, this);
    watchers.deploymentsDir?.onDidDelete(this.refreshDeployments, this);

    watchers.configurations?.onDidCreate(this.refreshConfigurations, this);
    watchers.configurations?.onDidDelete(this.refreshConfigurations, this);
    watchers.configurations?.onDidChange(this.refreshConfigurations, this);

    watchers.deployments?.onDidCreate(this.refreshDeployments, this);
    watchers.deployments?.onDidDelete(this.refreshDeployments, this);
    watchers.deployments?.onDidChange(this.refreshDeployments, this);

    watchers.allFiles?.onDidCreate(this.sendRefreshedFilesLists, this);
    watchers.allFiles?.onDidDelete(this.sendRefreshedFilesLists, this);
  }
}
