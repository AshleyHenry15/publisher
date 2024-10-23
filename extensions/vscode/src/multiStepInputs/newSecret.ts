import { window } from "vscode";

import { Configuration, useApi } from "src/api";
import {
  MultiStepInput,
  MultiStepState,
  assignStep,
  isQuickPickItem,
} from "src/multiStepInputs/multiStepHelper";
import { getSummaryStringFromError } from "src/utils/errors";
import { showProgress } from "src/utils/progress";

export async function newSecret(
  viewId: string,
  activeConfig: Configuration,
  callback: (name: string, value: string) => void,
): Promise<void> {
  const environment = activeConfig.configuration.environment;
  const existingKeys = new Set();
  if (environment) {
    for (const secret in environment) {
      existingKeys.add(secret);
    }
  }

  async function collectInputs() {
    const state: MultiStepState = {
      title: "Add a Secret",
      step: -1,
      lastStep: 0,
      totalSteps: 2,
      data: {
        name: <string | undefined>undefined,
        value: <string | undefined>undefined,
      },
      promptStepNumbers: {},
    };

    await MultiStepInput.run((input) => inputSecretName(input, state));
    return state;
  }

  async function inputSecretName(input: MultiStepInput, state: MultiStepState) {
    const step = assignStep(state, "inputSecretName");
    const currentName =
      typeof state.data.name === "string" ? state.data.name : "";

    const name = await input.showInputBox({
      title: state.title,
      step: step,
      totalSteps: state.totalSteps,
      value: currentName,
      prompt: "Enter the name of the secret",
      // eslint-disable-next-line require-await
      finalValidation: async (input: string) => {
        if (input.length === 0) {
          return "Secret names cannot be empty.";
        }
        if (existingKeys.has(input)) {
          return "There is already an environment variable with this name. Secrets and environment variable names must be unique.";
        }
        return;
      },
      shouldResume: () => Promise.resolve(false),
      ignoreFocusOut: true,
    });

    state.data.name = name;
    state.lastStep = step;
    return (input: MultiStepInput) => inputSecretValue(input, state);
  }

  async function inputSecretValue(
    input: MultiStepInput,
    state: MultiStepState,
  ) {
    const step = assignStep(state, "inputSecretValue");
    const currentValue =
      typeof state.data.value === "string" ? state.data.value : "";

    const value = await input.showInputBox({
      title: state.title,
      step: step,
      totalSteps: state.totalSteps,
      value: currentValue,
      prompt: "Enter the value of the secret",
      password: true,
      shouldResume: () => Promise.resolve(false),
      ignoreFocusOut: true,
    });

    state.data.value = value;
    state.lastStep = step;
  }

  const state = await collectInputs();
  const { name, value } = state.data;

  if (
    name === undefined ||
    isQuickPickItem(name) ||
    value === undefined ||
    isQuickPickItem(value)
  ) {
    // The user cancelled the input
    return;
  }

  try {
    await showProgress("Adding Secret", viewId, async () => {
      const api = await useApi();
      await api.secrets.add(
        activeConfig.configurationName,
        name,
        activeConfig.projectDir,
      );
    });
  } catch (error: unknown) {
    const summary = getSummaryStringFromError("addSecret", error);
    window.showInformationMessage(
      `Failed to add secret to configuration. ${summary}`,
    );
  }

  callback(name, value);
}
