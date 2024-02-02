## Publisher CLI

### `list-accounts`

This command pulls from both the `IDE` and `rsconnect-python` saved servers
list.

```
publisher list-accounts
```

### `init`

```
publisher init ${DIRECTORY}
```

The `init` command inspects the files in the specified project directory, or the
current directory if omitted. It creates an initial configuration file for you
in `.posit/publish/default.toml`. In the future we will support multiple
configuration files. For now, only `default`.

The client auto-detects the type of content in `DIRECTORY`.

For Python apps, `publisher` looks for a single Python source file in the
directory to be the entrypoint. If there are multiple python files, one of them
must be named `app.py`, `api.py`, `main.py`, or  `streamlit_app.py`.

For Quarto projects, `publisher` looks for a .qmd file with the same name as the
project directory, or for a single .qmd file to be the entrypoint.

Note: If the content type or entrypoint cannot be detected, the publisher sends
an error instructing you to manually update the default.toml file created.

Initialization also records the Python version and packages in use. You can
specify which Python interpreter to query with the `--python` option. You can
also provide a package list in `requirements.txt`. If you don't, one will be
created for you with `python -m pip freeze`; however, this is not recommended.

Currently, the UI/VSCode extension automatically runs `init` if there is no
configuration file. You can run `init` again if you need to, but note that it
will overwrite the default configuration file.

### `deploy`

The sample command below allows you to deploy a new content item in the Connect
account specified. Replace the `${ACCOUNT}` placeholder with a nickname of a
Connect account from the `list-accounts` command above.

Minimal sample command:

```
publisher deploy --name ${NAME} --account ${ACCOUNT} ${DIRECTORY}
```

or

```
publisher deploy -n ${NAME} --a ${ACCOUNT} ${DIRECTORY}
```

Creating a deployment publishes the contents of `DIRECTORY` to Connect, and
creates a record of the deployment in `.posit/publish/deployments/${NAME}.toml`
([schema](https://cdn.posit.co/publisher/schemas/posit-publishing-record-schema-v3.json)).
`DIRECTORY` is optional, and defaults to the current directory.


It is OK to rename deployment files as long as they live in
`.posit/publish/deployments` and have a `.toml` suffix. Deleting a deployment
file is also OK, and will cause `publisher` to forget about that deployment.
Neither action will cause a change to anything in Connect.

### `redeploy`

The sample command below allows you to redeploy a pre-existing, named
deployment.

```
publisher redeploy ${NAME} ${DIRECTORY}
```

`NAME` is the deployment name you provided in the `deploy` command.

`DIRECTORY` is optional, and defaults to the current directory.

## `publisher ui`

To launch the Posit Publisher UI in a browser, run the following command:
```
publisher ui ${DIRECTORY}
```
This returns an address you can visit or, if you include the `-i` flag, launches
an interactive browser session.

## help

Each `publisher` CLI command supports a `-h` option, which will show the help
for that command.

```
publisher ui -h
```