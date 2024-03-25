// Copyright (C) 2024 by Posit Software, PBC.

import {
  ExtensionContext,
  ProviderResult,
  ThemeIcon,
  TreeItem,
  Uri,
  commands,
  env,
} from "vscode";

import { PositTreeProvider } from "./toplevel";

const viewName = "posit.publisher.helpAndFeedback";
const openGettingStartedCommand = viewName + ".gettingStarted";
const openFeedbackCommand = viewName + "openFeedback";

export class HelpAndFeedbackTreeDataProvider
  implements PositTreeProvider<HelpAndFeedbackTreeItem>
{
  public name: string = "Help and Feedback";
  public iconPath: ThemeIcon = new ThemeIcon("info");

  constructor() {}

  getTreeItem(element: HelpAndFeedbackTreeItem): TreeItem | Thenable<TreeItem> {
    return element;
  }

  getChildren(
    element: HelpAndFeedbackTreeItem | undefined,
  ): ProviderResult<HelpAndFeedbackTreeItem[]> {
    if (element === undefined) {
      return [
        new HelpAndFeedbackTreeItem(
          "Get Started with Posit Publisher",
          "Open Getting Started Documentation",
          openGettingStartedCommand,
        ),
        new HelpAndFeedbackTreeItem(
          "Provide Feedback",
          "Open Feedback Slack Channel",
          openFeedbackCommand,
        ),
      ];
    }
    return [];
  }

  public register(context: ExtensionContext) {
    context.subscriptions.push(
      commands.registerCommand(openGettingStartedCommand, () => {
        env.openExternal(
          Uri.parse(
            "https://github.com/posit-dev/publisher/blob/e72828f3585497649b8b55470a665f7fa890a21f/docs/vscode.md",
          ),
        );
      }),
    );

    context.subscriptions.push(
      commands.registerCommand(openFeedbackCommand, () => {
        env.openExternal(
          Uri.parse("https://positpbc.slack.com/channels/publisher-feedback"),
        );
      }),
    );
  }
}

export class HelpAndFeedbackTreeItem extends TreeItem {
  constructor(itemString: string, commandTitle: string, command: string) {
    super(itemString);
    this.command = {
      title: commandTitle,
      command,
    };
  }
}
