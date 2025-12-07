# Manifest example

Gists can include a `gix.json` to tell gix exactly how to run instead of relying on shebangs or file extensions.

```json
{
  "run": "python app.py --verbose",
  "env": {
    "API_BASE": "https://api.example.com",
    "DEBUG": "1"
  }
}
```

You may include, for instance, `.sh` or `.bat` files that set up environment or perform other tasks before/after calling the main program to run. In the example below, `setup-and-run.sh` handles some setup before executing the gist.

```sh
{
  "run": "./setup-and-run.sh",
  "env": {
    "CONFIG_PATH": "/etc/myapp/config.yaml"
  }
}
```


- Place `gix.json` at the root of the gist. Override the filename with `--manifest <name>`.
- `run` is a single string and is executed via the shell (`sh -c` on Unix, `cmd /C` on Windows).
- `env` is optional; keys/values are injected for the run command.