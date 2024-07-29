import { browser } from "@wdio/globals";
import { exec } from "child_process";

const connectServer = process.env.CONNECT_SERVER;
const apiKey = process.env.CONNECT_API_KEY;

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

export async function openExtension() {
  browser.$("aria/Posit Publisher").waitForExist({ timeout: 30000 });

  // open posit extension
  const extension = await browser.$("aria/Posit Publisher");
  await expect(extension).toExist();
  await extension.click();
}

export async function firstDeployment(deploymentName: string) {
  let workbench: any;
  let input: any;
  workbench = await browser.getWorkbench();
  input = await $(".input");
  await switchToSubframe();
  // initialize project via button
  const init = await $('[data-automation="add-deployment-button"]');

  await expect(init).toHaveText("Add Deployment");
  await init.click();

  await browser.switchToFrame(null);

  // set title
  await input.setValue(deploymentName);
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
}

export function runShellScript(scriptPath: string) {
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

// export async function cleanup(contentFilePath: string) {
//   // const parentDir = path.resolve(
//   //     __dirname,
//   //     contentFilePath,
//   //   );
//   //   const positDir = path.join(parentDir, ".posit");

//   //   // Log the contents of the parent directory
//   //   console.log(fs.readdirSync(parentDir));

//   //   // Check if the directory exists before trying to delete it
//   //   if (fs.existsSync(positDir)) {
//   //     // Get the files in the directory
//   //     const files = fs.readdirSync(positDir);

//   //     // Delete each file in the directory
//   //     for (const file of files) {
//   //       const filePath = path.join(positDir, file);
//   //       if (fs.lstatSync(filePath).isDirectory()) {
//   //         fs.rmdirSync(filePath, { recursive: true }); // Delete directory recursively
//   //       } else {
//   //         fs.unlinkSync(filePath); // Delete file
//   //       }
//   //     }

//   //     // Delete the directory
//   //     fs.rmdirSync(positDir);
//   //   } else {
//   //     console.log("Directory does not exist");
//   //   }

//     // Use shell script to delete credentials
//   exec('"../../scripts/cleanup.bash" \\' + contentFilePath);
//   // const scriptPath = "./scripts/cleanup.bash " + contentFilePath;
//   // await runShellScript(scriptPath);
//   // Construct an absolute path to the script
//   // const scriptPath = path.join(__dirname, 'scripts', 'cleanup.bash');
//   // const command = `${scriptPath} ${contentFilePath}`;

//   // exec(command, (error, stdout, stderr) => {
//   // if (error) {
//   //     console.error(`exec error: ${error}`);
//   //     return;
//   // }
//   // console.log(`stdout: ${stdout}`);
//   // console.error(`stderr: ${stderr}`);
//   };
