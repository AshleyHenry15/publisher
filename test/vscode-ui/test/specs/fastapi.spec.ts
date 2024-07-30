import { browser, $, expect } from "@wdio/globals";

import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";
import { exec } from "child_process";
// import {
//   switchToSubframe,
//   waitForInputFields,
//   runShellScript,
// } from "../helpers.ts";

const connectServer = process.env.CONNECT_SERVER;
const apiKey = process.env.CONNECT_API_KEY;
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

async function switchToSubframe() {
  await browser.$(".webview");
  const iframe = await browser.$("iframe");
  await browser.switchToFrame(iframe);

  await browser.$("iframe").waitForExist({ timeout: 3000 });
  const subiframe = await browser.$("iframe");
  await subiframe.waitForExist({ timeout: 3000 });
  await browser.switchToFrame(subiframe);
}

async function waitForInputFields(inputText: string) {
  // wait until the server responds
  await browser.waitUntil(
    async () => {
      const element = await browser.$("#quickInput_message");
      const text = await element.getText();
      return text.includes(inputText);
    },
    {
      timeout: 30000,
      timeoutMsg:
        "Expected element signifying server response did not appear within timeout",
    },
  );
}

function runShellScript(scriptPath: string) {
  return new Promise((resolve, reject) => {
    exec(scriptPath, (error, stdout, stderr) => {
      if (error) {
        console.error(`exec error: ${error}`);
        return reject(error);
      }
      console.log(`stdout: ${stdout}`);
      console.error(`stderr: ${stderr}`);
      resolve(stdout);
    });
  });
}

describe("VS Code Extension UI Test", () => {
  let workbench: any;

  before(async () => {
    workbench = await browser.getWorkbench();
  });

  it("open extension", async () => {
    browser.$("aria/Posit Publisher").waitForExist({ timeout: 30000 });

    // open posit extension
    const extension = await browser.$("aria/Posit Publisher");
    await expect(extension).toExist();
    await extension.click();
  });

  it("can click add deployment button", async () => {
    await browser.pause(5000);
    await switchToSubframe();
    const addDeployBtn = await $('[data-automation="add-deployment-button"]');
    await expect(addDeployBtn).toHaveText("Add Deployment");
    await addDeployBtn.click();
  });

  it("can create deployment", async () => {
    let input: any;
    input = await $(".input");
    await browser.switchToFrame(null);

    // set title
    await input.setValue("my fastapi app");
    await browser.keys("\uE007");

    // set server url
    await input.setValue(connectServer);
    await browser.keys("\uE007");

    // wait until the server responds
    await waitForInputFields("The API key to be used");

    //set api key
    await input.setValue(apiKey);
    await browser.keys("\uE007");

    // wait for server validation
    await waitForInputFields("Enter a Unique Nickname");

    // set server name
    await input.setValue("my connect server");
    await browser.keys("\uE007");
  });

  it("check config", async () => {
    const workbench = await browser.getWorkbench();
    await expect(
      await workbench.getEditorView().getOpenEditorTitles(),
    ).toContain("configuration-1.toml");
    const filePath = path.resolve(
      __dirname,
      "../../../sample-content/fastapi-simple/.posit/publish/configuration-1.toml",
    );
    const fileContent = fs.readFileSync(filePath, "utf8");
    await expect(fileContent).toContain(
      "type = 'python-fastapi'\nentrypoint = 'simple.py'\nvalidate = true\nfiles = ['*']\ntitle = 'my fastapi app'",
    );
  });

  // cleanup
  after(async () => {
    const parentDir = path.resolve(
      __dirname,
      "../../../sample-content/fastapi-simple",
    );
    const positDir = path.join(parentDir, ".posit");

    // Log the contents of the parent directory
    console.log(fs.readdirSync(parentDir));

    // Check if the directory exists before trying to delete it
    if (fs.existsSync(positDir)) {
      // Get the files in the directory
      const files = fs.readdirSync(positDir);

      // Delete each file in the directory
      for (const file of files) {
        const filePath = path.join(positDir, file);
        if (fs.lstatSync(filePath).isDirectory()) {
          fs.rmdirSync(filePath, { recursive: true }); // Delete directory recursively
        } else {
          fs.unlinkSync(filePath); // Delete file
        }
      }

      // Delete the directory
      fs.rmdirSync(positDir);
    } else {
      console.log("Directory does not exist");
    }

    // Use shell script to delete credentials
    describe("Cleanup creds", () => {
      it("remove credentials", async () => {
        const scriptPath =
          "../scripts/cleanup.bash ../../sample-content/fastapi-simple";
        await runShellScript(scriptPath);
      });
    });
  });
});
