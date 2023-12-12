
import * as retry from 'retry';
import * as vscode from 'vscode';

import * as commands from './commands';
import * as ports from './ports';
import { Panel } from './panels';

export class Assistant {

    private readonly name: string = "Publisher";

	private readonly panel: Panel;
	private readonly path: string;
	private readonly port: number;
	private readonly terminal: vscode.Terminal;

	constructor (panel: Panel, path: string, port: number) {
		this.panel = panel;
		this.path = path;
		this.port = port;
		this.terminal = vscode.window.createTerminal({ name: this.name, hideFromUser: true });
	}

	show = async () => {
		return this.panel.show();
	};

	start = async (): Promise<void> => {
		const command: commands.Command = commands.create(this.path, this.port);
		this.terminal.sendText(command);
		if (!(await ports.ping(this.port))) {
			throw Error("publisher failed to start");
		}
	};

	stop = async (): Promise<void> => {
		// close the panel
		this.panel.dispose();
		const operation = retry.operation();
		operation.attempt(async () => {
			// send "CTRL+C" command
			this.terminal.sendText("\u0003");
			const pong = await ports.ping(this.port, 1000);
			if (pong) {
				throw Error("application is still running");
			}
		});
	};
}
