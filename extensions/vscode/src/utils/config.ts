import { Uri, commands, workspace } from "vscode";
import { fileExists, isDir } from "./files";
import { substituteVariables } from "./variables";

export async function getPythonInterpreterPath(): Promise<string | undefined> {
  const workspaceFolder = workspace.workspaceFolders?.[0];
  if (workspaceFolder === undefined) {
    return undefined;
  }
  const configuredPython = await commands.executeCommand<string>(
    "python.interpreterPath",
    { workspaceFolder: workspaceFolder },
  );
  if (configuredPython === undefined) {
    return undefined;
  }
  let python = substituteVariables(configuredPython, true);
  const pythonUri = Uri.file(python);

  if (await isDir(pythonUri)) {
    // Configured python can be a directory such as a virtual environment.
    const names = [
      "bin/python",
      "bin/python3",
      "Scripts/python.exe",
      "Scripts/python3.exe",
    ];
    for (let name of names) {
      const candidate = Uri.joinPath(pythonUri, name);
      if (await fileExists(candidate)) {
        python = candidate.fsPath;
      }
    }
  }
  console.log("Python interpreter path:", python);
  return python;
}
