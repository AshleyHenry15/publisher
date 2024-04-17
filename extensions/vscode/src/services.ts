// Copyright (C) 2024 by Posit Software, PBC.

import { HOST } from ".";
import { Server } from "./servers";
import { useApi } from "./api";
import { ExtensionContext, Disposable } from "vscode";

export class Service implements Disposable {
  private context: ExtensionContext;
  private server: Server;
  private agentURL: string;

  constructor(context: ExtensionContext, port: number) {
    this.context = context;
    this.agentURL = `http://${HOST}:${port}/api`;
    this.server = new Server(port);
    useApi(this.agentURL, this.isUp());
  }

  start = async () => {
    await this.server.start(this.context);
  };

  isUp = () => {
    return this.server.isUp();
  };

  stop = async () => {
    await this.server.stop();
    this.server.dispose();
  };

  dispose() {
    this.server.dispose();
  }
}
