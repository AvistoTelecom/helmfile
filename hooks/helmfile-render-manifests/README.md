# helmfile-render-manifests

pre-commit hook that runs the **regular helmfile CLI** with an embedded helm (Helm v4 SDK, no helm
or helmfile install needed), writes its stdout to a file, and fails when the file was stale or the
output contains a `kind: Secret`.

Hook flags, then `--`, then any normal helmfile arguments (nothing is re-implemented):

```yaml
- repo: https://github.com/AvistoTelecom/helmfile
  rev: <sha or tag>
  hooks:
    - id: render-manifests
      alias: render-int
      args: [--output=int/manifests/manifests.yaml, --, -f, helmfile.yaml.gotmpl, -e, int, template]
```

| Hook flag | Default | |
|---|---|---|
| `--output` | required | file receiving helmfile's stdout |
| `--check` | false | don't write, only fail if stale |
| `--fail-on-secret` | true | fail on `kind: Secret` |

One hook entry per output file (e.g. per environment). Rendering ignores any kube context
(`KUBECONFIG=/dev/null`) so the output matches a CI render.
