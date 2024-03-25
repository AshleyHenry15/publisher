// Copyright (C) 2024 by Posit Software, PBC.

import {
  TreeItem,
  ProviderResult,
  ExtensionContext,
} from "vscode";

import { PositTreeProvider } from "./toplevel";

// const viewName = "posit.publisher.project";

export class ProjectTreeDataProvider
  implements PositTreeProvider<ProjectTreeItem>
{
  public name: string = "Project";
  public iconPath = undefined;

  constructor() {}

  getTreeItem(element: ProjectTreeItem): TreeItem | Thenable<TreeItem> {
    return element;
  }

  getChildren(
    _: ProjectTreeItem | undefined,
  ): ProviderResult<ProjectTreeItem[]> {
    return [];
  }

  public register(_: ExtensionContext) {}
}

export class ProjectTreeItem extends TreeItem {
  constructor(itemString: string) {
    super(itemString);
  }

  contextValue = "posit.publisher.project.tree.item";
  tooltip = "This is a \nProject Tree Item";
}
