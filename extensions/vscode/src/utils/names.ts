// Copyright (C) 2024 by Posit Software, PBC.

import { InputBoxValidationSeverity } from "vscode";

import { useApi } from "src/api";
import { isValidFilename } from "src/utils/files";

export async function untitledConfigurationName(
  startingId?: number,
): Promise<string> {
  const api = await useApi();
  const existingConfigurations = (await api.configurations.getAll()).data;

  let id = startingId || new Date().getTime();
  let defaultName = "";
  do {
    const trialName = `configuration-${id}`;

    if (
      !existingConfigurations.find((config) => {
        return (
          config.configurationName.toLowerCase() === trialName.toLowerCase()
        );
      })
    ) {
      defaultName = trialName;
    }
    id += 1;
  } while (!defaultName);
  return defaultName;
}

export function untitledContentRecordName(
  existingContentRecordNames: string[],
  startingId?: number,
): string {
  let id = startingId || new Date().getTime();
  let defaultName = "";
  do {
    const trialName = `deployment-${id}`;

    if (uniqueContentRecordName(trialName, existingContentRecordNames)) {
      defaultName = trialName;
    }
    id += 1;
  } while (!defaultName);
  return defaultName;
}

export function uniqueContentRecordName(
  nameToTest: string,
  existingNames: string[],
) {
  return !existingNames.find((existingName) => {
    return existingName.toLowerCase() === nameToTest.toLowerCase();
  });
}

export function contentRecordNameValidator(
  contentRecordNames: string[],
  currentName: string,
) {
  return async (value: string) => {
    const isUnique =
      value === currentName ||
      uniqueContentRecordName(value, contentRecordNames);

    if (value.length < 3 || !isUnique || !isValidFilename(value)) {
      return {
        message: `Invalid Name: Value must be unique across other deployment record names for this project, be longer than 3 characters, cannot be '.' or contain '..' or any of these characters: /:*?"<>|\\`,
        severity: InputBoxValidationSeverity.Error,
      };
    }
    return undefined;
  };
}
