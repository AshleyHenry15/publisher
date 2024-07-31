import { browser } from "@wdio/globals";
// import { exec } from "child_process";
import * as shell from "shelljs";

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

// export function runShellScript() {
//   return new Promise((resolve, reject) => {
//     const command = `
//       CREDS_GUID="$(${process.env.EXE} credentials list | jq -r '.[] | select(.name == "my connect server") | .guid')"
//       ${process.env.EXE} credentials delete $CREDS_GUID
//     `;
//     exec(command, (error, stdout, stderr) => {
//       if (error) {
//         console.error(`exec error: ${error}`);
//         return reject(error);
//       }
//       console.log(`stdout: ${stdout}`);
//       console.error(`stderr: ${stderr}`);
//       resolve(stdout);
//     });
//   });
// }

// const shell = require("shelljs");

export function runShellScript() {
  return new Promise((resolve, reject) => {
    shell.env["CREDS_GUID"] = shell
      .exec(
        `${process.env.EXE} credentials list | jq -r '.[] | select(.name == "my connect server") | .guid'`,
        { silent: true },
      )
      .stdout.trim();

    if (shell.env["CREDS_GUID"]) {
      const command = `
        ${process.env.EXE} credentials delete ${shell.env["CREDS_GUID"]}
      `;
      shell.exec(
        command,
        { silent: true },
        (code: number, stdout: string, stderr: string) => {
          if (code !== 0) {
            console.error(`exec error: ${stderr}`);
            return reject(new Error(stderr));
          }
          console.log(`stdout: ${stdout}`);
          console.error(`stderr: ${stderr}`);
          resolve(stdout);
        },
      );
    } else {
      reject(new Error("Failed to retrieve CREDS_GUID"));
    }
  });
}
