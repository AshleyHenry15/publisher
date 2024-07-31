import { browser } from "@wdio/globals";
import { exec } from "child_process";

export async function switchToSubframe() {
  await browser.$(".webview");
  const iframe = await browser.$("iframe");
  await browser.switchToFrame(iframe);

  await browser.$("iframe").waitForExist({ timeout: 3000 });
  const subiframe = await browser.$("iframe");
  await subiframe.waitForExist({ timeout: 3000 });
  await browser.switchToFrame(subiframe);
}

export async function waitForInputFields(inputText: string) {
  // wait until the server responds
  await browser.waitUntil(
    async () => {
      const element = await browser.$("#quickInput_message");
      const text = await element.getText();
      return text.includes(inputText);
    },
    {
      timeout: 7000, // Timeout in milliseconds, adjust as necessary
      timeoutMsg:
        "Expected element signifying server response did not appear within timeout",
    },
  );
}

export async function runShellScript(scriptPath: string, args: string) {
  return new Promise((resolve, reject) => {
    const command =
      process.platform === "win32"
        ? `sh ${scriptPath} ${args}`
        : `bash ${scriptPath} ${args}`;
    exec(command, (error, stdout, stderr) => {
      if (error) {
        reject(`Error executing script: ${error.message}`);
        return;
      }
      if (stderr) {
        reject(`Script stderr: ${stderr}`);
        return;
      }
      resolve(stdout);
    });
  });
}
