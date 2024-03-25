// Copyright (C) 2024 by Posit Software, PBC.

import {
  commands,
  Event,
  EventEmitter,
  ExtensionContext,
  ThemeIcon,
  TreeDataProvider,
  TreeItem,
  TreeItemCollapsibleState,
  window,
} from "vscode";

const viewName = "posit.publisher.toplevel";
const refreshCommand = viewName + ".refresh";

type ConfigurationEventEmitter = EventEmitter<
TopLevelTreeItem | undefined | void
>;
type ConfigurationEvent = Event<TopLevelTreeItem | undefined | void>;


export class TopLevelTreeProvider
  implements TreeDataProvider<TreeItem>
{
  private providerMap = new Map<TreeItem, PositTreeProvider<TreeItem>>;
  private providerChildMap = new Map<TreeItem, PositTreeProvider<TreeItem>>;
  private _onDidChangeTreeData: ConfigurationEventEmitter = new EventEmitter();
  readonly onDidChangeTreeData: ConfigurationEvent =
    this._onDidChangeTreeData.event;

  constructor(providers: PositTreeProvider<TreeItem>[]) {
    providers.forEach(p => {
      const item = new TopLevelTreeItem(p.name, p.iconPath);
      this.providerMap.set(item, p);
    });
  }

  getTreeItem(element: TreeItem): TreeItem | Thenable<TreeItem> {
    return element;
  }
  private refresh = () => {
    this._onDidChangeTreeData.fire();
  };

  async getChildren(
    parent: TopLevelTreeItem | undefined,
  ): Promise<TreeItem[] | null | undefined> {
    if (parent === undefined) {
      const children = new Array<TreeItem>;
      this.providerMap.forEach((_, child) => children.push(child));
      return children;
    } else {
      const provider = this.providerMap.get(parent) || this.providerChildMap.get(parent);
      if (provider === undefined) {
        throw new Error("undefined provider");
      }

      // If this is one of our items, call the provider with undefined
      // to get their top level items.
      let item = undefined;
      if (!parent.contextValue?.startsWith("posit.publisher.toplevel.")) {
        item = parent;
      }
      const children = await provider.getChildren(item);
      if (children) {
        children.forEach(c => this.providerChildMap.set(c, provider));
      }
      return children;
    }
  }

  public register(context: ExtensionContext) {
    context.subscriptions.push(
      window.createTreeView(viewName, { treeDataProvider: this }),
      commands.registerCommand(refreshCommand, this.refresh),
    );
  }
}

export interface PositTreeProvider<T> extends TreeDataProvider<T> {
  name: string;
  iconPath: ThemeIcon | undefined;
}

export class TopLevelTreeItem extends TreeItem {
  constructor(itemString: string, iconPath: ThemeIcon | undefined) {
    super(itemString);
    this.iconPath = iconPath;
    this.collapsibleState = TreeItemCollapsibleState.Collapsed;
    this.contextValue = "posit.publisher.toplevel." + itemString.toLowerCase();
  }

  tooltip = "";
}
