import * as vscode from 'vscode';

import { HOST } from '.';

const EXECUTABLE_DEFAULT = "publisher";

export type Command = string;

export const create = (path: string, port: number, subcommand: string = "ui"): Command => {
    const configuration =  vscode.workspace.getConfiguration('posit');
    let executable: string = configuration.get<string>('publisher.executable.path', EXECUTABLE_DEFAULT);
    if (!executable) {
        executable = EXECUTABLE_DEFAULT;
    }

    return `${executable} ${subcommand} -v --listen=${HOST}:${port} ${path}`;
};
