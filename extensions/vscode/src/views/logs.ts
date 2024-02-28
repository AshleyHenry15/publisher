// Copyright (C) 2024 by Posit Software, PBC.

import * as vscode from 'vscode';

import { EventStream, EventStreamMessage, displayEventStreamMessage } from '../events';

const viewName = 'posit.publisher.logs';

/**
 * Tree data provider for the Logs view.
 */
export class LogsTreeDataProvider implements vscode.TreeDataProvider<LogsTreeItem> {
  private events: EventStreamMessage[] = [];

  /**
   * Event emitter for when the tree data of the Logs view changes.
   * @private
   */
  private _onDidChangeTreeData: vscode.EventEmitter<LogsTreeItem | undefined> = new vscode.EventEmitter<LogsTreeItem | undefined>();

  /**
   * Creates an instance of LogsTreeDataProvider.
   * @constructor
   * @param {EventStream} stream - The event stream to listen to.
   */
  constructor(stream: EventStream) {
    stream.on('message', (msg: EventStreamMessage) => {
      if (msg.data.level !== 'DEBUG' && msg.type !== 'agent/log') {
        this.events.push(msg);
        this.refresh();
      }
    });

    // Reset events when a new publish starts
    stream.register('publish/start', (_: EventStreamMessage) => {
      this.events = [];
    });
  };

  /**
   * Get the event emitter for tree data changes.
   */
  get onDidChangeTreeData(): vscode.Event<LogsTreeItem | undefined> {
    return this._onDidChangeTreeData.event;
  }

  /**
   * Refresh the tree view by firing the onDidChangeTreeData event.
   */
  public refresh(): void {
    this._onDidChangeTreeData.fire(undefined);
  }

  /**
   * Get the tree item for the specified element.
   * @param element The element for which to get the tree item.
   * @returns The tree item representing the element.
   */
  getTreeItem(element: LogsTreeItem): vscode.TreeItem | Thenable<vscode.TreeItem> {
    return element;
  }

  /**
   * Get the child elements of the specified element.
   * @param _ The parent element.
   * @returns The child elements of the parent element.
   */
  getChildren(_: LogsTreeItem | undefined): vscode.ProviderResult<LogsTreeItem[]> {
    // Map the events array to LogsTreeItem instances and return them as children
    return this.events.map((event) => new LogsTreeItem(event, vscode.TreeItemCollapsibleState.None));
  }

  /**
   * Register the tree view in the extension context.
   * @param context The extension context.
   */
  public register(context: vscode.ExtensionContext) {
    // Register the tree data provider
    vscode.window.registerTreeDataProvider(viewName, this);
    // Create a tree view with the specified view name and options
    context.subscriptions.push(
      vscode.window.createTreeView(viewName, {
        treeDataProvider: this
      })
    );
  }
}

/**
 * Represents a tree item for displaying logs in the tree view.
 */
export class LogsTreeItem extends vscode.TreeItem {
  constructor(msg: EventStreamMessage, state: vscode.TreeItemCollapsibleState = vscode.TreeItemCollapsibleState.None) {
    super(displayEventStreamMessage(msg), state);
    this.tooltip = JSON.stringify(msg);
  }
}