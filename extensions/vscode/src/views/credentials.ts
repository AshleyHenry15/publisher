import {
  TreeDataProvider,
  TreeItem,
  ExtensionContext,
  window,
} from 'vscode';

import api, { Account } from '../api';

const viewName = 'posit.publisher.credentials';

export class CredentialsTreeDataProvider implements TreeDataProvider<CredentialsTreeItem> {

  constructor() { }

  getTreeItem(element: CredentialsTreeItem): TreeItem | Thenable<TreeItem> {
    return element;
  }

  async getChildren(element: CredentialsTreeItem | undefined): Promise<CredentialsTreeItem[]> {
    if (element) {
      return [];
    }

    const response = await api.accounts.getAll();
    const accounts = response.data.accounts;
    return accounts.map(account => {
      return new CredentialsTreeItem(account);
    });
  }

  public register(context: ExtensionContext) {
    window.registerTreeDataProvider(viewName, this);
    context.subscriptions.push(
      window.createTreeView(viewName, { treeDataProvider: this })
    );
  }
}
export class CredentialsTreeItem extends TreeItem {
  contextValue = 'posit.publisher.credentials.tree.item';

  constructor(account: Account) {
    super(account.name);
    this.tooltip = this.getTooltip(account);
  }

  getTooltip(account: Account): string {
    let result = '';

    if (account.authType === 'token-key') {
      result += `Account: ${account.accountName}\n`;
    } else if (account.authType === 'api-key') {
      result += 'Account: Using API Key\n';
    }

    result += `URL: ${account.url}\n`;
    result += `Managed by: ${account.source}`;

    return result;
  }
}
