// Copyright (C) 2024 by Posit Software, PBC.

import { ProgressLocation, Uri, commands, env, window } from "vscode";
import { eventTypeToString } from "../api";
import { EventStream, EventStreamMessage, UnregisterCallback } from "../events";

const showPublisherLogsCommand =
  "workbench.action.output.show.extension-output-posit.publisher-#1-Posit Publisher";
const showDeploymentLogsCommand = "posit.publisher.logs.focus";

export function deployProject(localID: string, stream: EventStream) {
  window.withProgress(
    {
      location: ProgressLocation.Notification,
      title: `Deploying your project...`,
      cancellable: false,
    },
    (progress) => {
      let resolveCB: (reason?: any) => void;
      let rejectCB: (reason?: any) => void;
      const registrations: UnregisterCallback[] = [];

      const unregiserAll = () => {
        registrations.forEach((cb) => cb.unregister());
      };

      const promise = new Promise<void>((resolve, reject) => {
        resolveCB = resolve;
        rejectCB = reject;
      });

      registrations.push(
        stream.register("publish/start", (msg: EventStreamMessage) => {
          if (localID === msg.data.localId) {
            progress.report({ message: "Starting to Deploy..." });
          }
        }),
      );

      const handleProgressMessages = (msg: EventStreamMessage) => {
        if (localID === msg.data.localId) {
          const progressStr = eventTypeToString(msg.type);
          progress.report({
            message: progressStr,
          });
          console.log(progressStr);
        }
      };

      registrations.push(
        stream.register(
          "publish/checkCapabilities/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/checkCapabilities/log",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/checkCapabilities/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/checkCapabilities/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createNewDeployment/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createNewDeployment/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createNewDeployment/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/setEnvVars/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/setEnvVars/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/setEnvVars/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createBundle/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createBundle/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createBundle/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createBundle/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createDeployment/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createDeployment/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createDeployment/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/createDeployment/log",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/uploadBundle/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/uploadBundle/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/uploadBundle/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/uploadBundle/log",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/deployBundle/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/deployBundle/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/deployBundle/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/deployBundle/log",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/restorePythonEnv/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/restorePythonEnv/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/restorePythonEnv/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/restorePythonEnv/log",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/restorePythonEnv/progress",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/restorePythonEnv/status",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/runContent/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/runContent/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/runContent/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register("publish/runContent/log", (msg: EventStreamMessage) => {
          handleProgressMessages(msg);
        }),
      );
      registrations.push(
        stream.register(
          "publish/setVanityURL/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/setVanityURL/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/setVanityURL/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/setVanityURL/log",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/validateDeployment/start",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/validateDeployment/success",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/validateDeployment/failure",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );
      registrations.push(
        stream.register(
          "publish/validateDeployment/log",
          (msg: EventStreamMessage) => {
            handleProgressMessages(msg);
          },
        ),
      );

      registrations.push(
        stream.register("publish/success", async (msg: EventStreamMessage) => {
          if (localID === msg.data.localId) {
            unregiserAll();
            progress.report({
              message: "Deployment was successful",
            });
            resolveCB("Success!");

            let visitOption = "Visit";
            const selection = await window.showInformationMessage(
              "Deployment was successful",
              visitOption,
            );
            if (selection === visitOption) {
              const uri = Uri.parse(msg.data.dashboardUrl, true);
              await env.openExternal(uri);
            }
          }
        }),
      );

      registrations.push(
        stream.register("publish/failure", async (msg: EventStreamMessage) => {
          if (localID === msg.data.localId) {
            unregiserAll();
            progress.report({
              message: "Deployment process encountered an error",
            });
            rejectCB("Error Encountered!");

            let deploymentLogsOption = "Deployment Logs";
            let debugLogsOption = "Debug Logs";
            const selection = await window.showInformationMessage(
              "Deployment process encountered an error",
              deploymentLogsOption,
              debugLogsOption,
            );
            switch (selection) {
              case deploymentLogsOption:
                await commands.executeCommand(showDeploymentLogsCommand);
                break;
              case debugLogsOption:
                await commands.executeCommand(showPublisherLogsCommand);
                break;
            }
          }
        }),
      );

      return promise;
    },
  );
}
