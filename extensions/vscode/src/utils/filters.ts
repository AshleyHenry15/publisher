// Copyright (C) 2024 by Posit Software, PBC.

import {
  Configuration,
  ConfigurationDetails,
  ConfigurationError,
  ContentType,
  isConfigurationError,
} from "../api";

export function filterConfigurationDetailsToType(
  configDetails: ConfigurationDetails[],
  type: ContentType | undefined,
): ConfigurationDetails[] {
  if (!type || type === ContentType.UNKNOWN) {
    return configDetails;
  }
  return configDetails.filter((c) => isConfigurationDetailsOfType(c, type));
}

export function isConfigurationDetailsOfType(
  configDetails: ConfigurationDetails,
  type?: ContentType,
): boolean {
  if (type === undefined) {
    return false;
  }
  return configDetails.type === type;
}

export function filterConfigurationsToValidAndType(
  configs: (Configuration | ConfigurationError)[],
  type: ContentType | undefined,
): Configuration[] {
  let result = configs.filter(
    (c): c is Configuration => !isConfigurationError(c),
  );
  if (type && type !== ContentType.UNKNOWN) {
    result = result.filter((c) => isConfigurationOfType(c, type));
  }
  return result;
}

export function isConfigurationOfType(
  config: Configuration,
  type?: ContentType,
): boolean {
  if (type === undefined) {
    return false;
  }
  return config.configuration.type === type;
}
