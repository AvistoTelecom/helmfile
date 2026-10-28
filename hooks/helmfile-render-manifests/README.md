# helmfile-render-manifests

pre-commit hook that renders helmfile environments to files and fails when they are stale or
contain a `kind: Secret`. Helm is embedded (Helm v4 SDK): nothing but Go-less pre-commit is needed.

```yaml
- repo: https://github.com/AvistoTelecom/helmfile
  rev: <sha or tag>
  hooks:
    - id: render-manifests
      args: [--file=helmfile.yaml.gotmpl, -e, int, -e, preprod]
```

| Flag | Default | |
|---|---|---|
| `--environment`, `-e` | required, repeatable | environments to render |
| `--file` | helmfile lookup | helmfile path |
| `--output` | `{env}/manifests/manifests.yaml` | output path, `{env}` = environment |
| `--check` | false | don't write, only fail if stale |
| `--fail-on-secret` | true | fail on `kind: Secret` |
| `--kube-version`, `--include-crds`, `--skip-tests`, `--state-values-set` | | forwarded to `helmfile template` |

Extra arguments after `--` go to `helmfile template`. Rendering ignores any kube context
(`KUBECONFIG=/dev/null`) so the output matches a CI render.
