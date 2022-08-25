# Git hooks

The repository offers git hooks to run lints and checks locally before committing changes. Under the hood this uses
[pre-commit](https://pre-commit.com/) plugins.

To install the git hooks, run the following:
```shell
$ make setup/git/hooks
```

Currently, the following plugins are enabled:
- [detect-secrets](https://github.com/Yelp/detect-secrets): detect secrets to avoid accidentially commit them.
- [golangci-lint](https://github.com/golangci/golangci-lint): run golangci-lint for changed go files.
- [trailing-whitespace](https://github.com/pre-commit/pre-commit-hooks#trailing-whitespace): detect trailing whitespaces.
- [end-of-file-fixer](https://github.com/pre-commit/pre-commit-hooks#end-of-file-fixer): ensure files end in a newline.
- [check-json](https://github.com/pre-commit/pre-commit-hooks#check-json): ensure all JSON files have valid syntax.

**Troubleshooting:**
In case you run into any issues with the secrets, the output typically is enough to highlight the problem.
However, in the case of the `detect-secrets`, you are required to do a few more steps:
```shell
# Ensure you have detect-secrets installed
$ pip3 install detect-secrets

# In the repository root, scan the repo for secrets and update the .secrets.baseline file.
$ detect-secrets scan --baseline .secrets.baseline

# Trigger an interactive audit of newly detected secrets. You will be asked to verify whether the detected secret
# is safe to be committed to the repository or not.
$ detect-secrets audit .secrets.baseline

# Afterwards, make sure you commit the updated .secrets.baseline file and the plugin should not return any errors
# anymore!

# To exclude a false positive secret use:
// pragma: allowlist secret
```
