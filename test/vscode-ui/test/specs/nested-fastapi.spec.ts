import { browser, $ } from "@wdio/globals";

import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";
import {
  openExtension,
  runShellScript,
  switchToSubframe,
  waitForInputFields,
} from "../helpers.ts";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const connectServer = process.env.CONNECT_SERVER;
const apiKey = process.env.CONNECT_API_KEY;

describe("Nested Fast API Deployment", () => {
  let workbench: any;
  let input: any;

  before(async () => {
    workbench = await browser.getWorkbench();
    input = await $(".input");
  });

  it("open extension", async () => {
    await openExtension();
  });

  it("can add deployment", async () => {
    await browser.pause(5000);
    await switchToSubframe();
    const addDeployBtn = await $('[data-automation="add-deployment-button"]');
    await addDeployBtn.click();
  });

  it("can list each entrypoint", async () => {
    await browser.switchToFrame(null);

    // verify each entrypoint is found and listed
    const quickpick = await browser.$(".quick-input-list");
    await quickpick.waitForExist({ timeout: 30000 });

    const simplepy = await browser.$(
      "aria/simple.py, (run with FastAPI), fastapi-simple/",
    );
    expect(simplepy).toExist();

    const quartoProjNoneMulti = await browser.$(
      "aria/quarto-proj-none.qmd, (render with Quarto), multi-type/",
    );
    expect(quartoProjNoneMulti).toExist();

    const simplepyMulti = await browser.$(
      "aria/simple.py, (run with FastAPI), multi-type/",
    );
    expect(simplepyMulti).toExist();

    const quartoProjNone = await browser.$(
      "aria/quarto-proj-none.qmd, (render with Quarto), quarto-proj-none/",
    );
    expect(quartoProjNone).toExist();

    const quartoProjPy = await browser.$(
      "aria/quarto-proj-py.qmd, (render with Quarto), quarto-proj-py/",
    );
    expect(quartoProjPy).toExist();

    const quartoProjR = await browser.$(
      "aria/quarto-proj-r.qmd, (render with Quarto), quarto-proj-r/",
    );
    expect(quartoProjR).toExist();

    const quartoProject = await browser.$(
      "aria/quarto-project.qmd, (render with Quarto), quarto-project/",
    );
    expect(quartoProject).toExist();

    const rmdHtml = await browser.$(
      "aria/index.htm, (serve pre-rendered HTML), rmd-static-1/",
    );
    expect(rmdHtml).toExist();

    const rmdKnitr = await browser.$(
      "aria/static.Rmd, (render with rmarkdown/knitr), rmd-static-1/",
    );
    expect(rmdKnitr).toExist();

    const rmdQuarto = await browser.$(
      "aria/static.Rmd, (render with Quarto), rmd-static-1/",
    );
    expect(rmdQuarto).toExist();

    const shiny = await browser.$("aria/app.R, (run with R Shiny), shinyapp/");
    expect(shiny).toExist();

    const shinyHtml = await browser.$(
      "aria/index.htm, (serve pre-rendered HTML), shinyapp/",
    );
    expect(shinyHtml).toExist();
  });

  it("can select entrypoint", async () => {
    const simplepy = await browser.$(
      "aria/simple.py, (run with FastAPI), fastapi-simple/",
    );
    const input = await $(".input");
    await expect(simplepy).toExist();
    await simplepy.click();

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

  it("can check config", async () => {
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
