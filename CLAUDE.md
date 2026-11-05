# isom-maplibre: agent instructions

## Local validation

GitHub Actions CI is off to save paid minutes, so nothing checks a pull request on GitHub. Run `scripts/check.sh` and make it pass before pushing.

- `scripts/check.sh` runs what the test job of `.github/workflows/release-please.yml` ran: go vet, go test, `go generate` with no change under `src/`, then `npm ci`, `npm run build` and `npm test`. It takes under a minute, so it has no `--full` mode.
- The pre-push hook (`.githooks/pre-push`) runs `scripts/check.sh`. Claude Code sessions turn it on at start; humans turn it on once with `git config core.hooksPath .githooks`. `git push --no-verify` skips it.
