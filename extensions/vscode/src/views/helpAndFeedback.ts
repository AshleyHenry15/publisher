// Copyright (C) 2024 by Posit Software, PBC.

import { Commands, Views } from "src/constants";
import {
  TreeDataProvider,
  TreeItem,
  ProviderResult,
  ExtensionContext,
  window,
  Uri,
  commands,
  env,
} from "vscode";

export class HelpAndFeedbackTreeDataProvider
  implements TreeDataProvider<HelpAndFeedbackTreeItem>
{
  constructor(private readonly context: ExtensionContext) {}

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
          Commands.HelpAndFeedback.OpenGettingStarted,
        ),
        new HelpAndFeedbackTreeItem(
          "Provide Feedback",
          "Open GitHub Discussions",
          Commands.HelpAndFeedback.OpenFeedback,
        ),
      ];
    }
    return [];
  }

  public register() {
    this.context.subscriptions.push(
      window.createTreeView(Views.HelpAndFeedback, { treeDataProvider: this }),
    );

    this.context.subscriptions.push(
      commands.registerCommand(
        Commands.HelpAndFeedback.OpenGettingStarted,
        () => {
          env.openExternal(
            Uri.parse(
              "https://github.com/posit-dev/publisher/blob/main/docs/index.md",
            ),
          );
        },
      ),
    );

    this.context.subscriptions.push(
      commands.registerCommand(Commands.HelpAndFeedback.OpenFeedback, () => {
        env.openExternal(
          Uri.parse("https://github.com/posit-dev/publisher/discussions"),
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
