# Sovereign brotli deploy

An **additive** mirror of the GitHub Pages site (https://go-widgets.github.io/gallery/)
that serves the wasm demos from our self-hosted Caddy with **native brotli**
(`Content-Encoding: br`) — ~24% smaller on the wire than Pages' gzip, decoded
natively by the browser (no client-side decoder). Pages stays as-is.

- **Host:** the Forgejo VM `157.136.250.7` (Debian 13), Caddy at
  `/etc/caddy/Caddyfile` (already serving `sources.mesocentre.plateau-de-saclay.net`).
- **Workflow:** `.github/workflows/deploy-sovereign.yml` — builds `gallery.wasm`
  + `iso.wasm` exactly like `pages.yml`, stamps `__WASM_BYTES__`, generates
  `.wasm.br` (brotli -11) + `.wasm.gz` (gzip -9) sidecars, and `rsync`s the site
  to the VM. It is **inert** until `SOVEREIGN_DEPLOY_ENABLED == 'true'`.

## One-time setup

1. **DNS** — add an `A` record for your chosen name (e.g.
   `widgets.mesocentre.plateau-de-saclay.net`) → `157.136.250.7`.
   Caddy gets the Let's Encrypt cert automatically on first request.

2. **Deploy key** — on the VM, create an unprivileged deploy user + dir:
   ```sh
   sudo useradd -m -s /usr/sbin/nologin gowdeploy || true
   sudo mkdir -p /var/www/gowidgets-gallery
   sudo chown gowdeploy:gowdeploy /var/www/gowidgets-gallery
   sudo -u gowdeploy mkdir -p /home/gowdeploy/.ssh && sudo -u gowdeploy chmod 700 /home/gowdeploy/.ssh
   ```
   Generate a keypair (`ssh-keygen -t ed25519 -f gowdeploy_key -N ''`), append
   the **public** key to `/home/gowdeploy/.ssh/authorized_keys`, and give Caddy
   read access to the web root (it runs as the `caddy` user):
   `sudo setfacl -R -m u:caddy:rX /var/www/gowidgets-gallery` (or make the dir
   group-readable).

3. **GitHub repo config** (Settings → Secrets and variables → Actions):
   - Secret `SOVEREIGN_DEPLOY_KEY` = the **private** deploy key.
   - Variable `SOVEREIGN_DEPLOY_HOST` = `157.136.250.7`
   - Variable `SOVEREIGN_DEPLOY_USER` = `gowdeploy`
   - Variable `SOVEREIGN_DEPLOY_PATH` = `/var/www/gowidgets-gallery`
   - Variable `SOVEREIGN_DEPLOY_KNOWN_HOSTS` = the VM's SSH host key line
     (pin it: `ssh-keyscan 157.136.250.7`) — recommended over trust-on-first-use.
   - Variable `SOVEREIGN_DEPLOY_ENABLED` = `true` (flips the workflow on).

4. **Caddy** — add `deploy/Caddyfile.snippet` (hostname adjusted) to
   `/etc/caddy/Caddyfile`, then:
   ```sh
   sudo cp /etc/caddy/Caddyfile /etc/caddy/Caddyfile.bak-pregallery
   sudoedit /etc/caddy/Caddyfile   # paste the block
   sudo caddy validate --config /etc/caddy/Caddyfile
   sudo systemctl reload caddy
   ```

## Verify

```sh
curl -sI -H 'Accept-Encoding: br' https://widgets.mesocentre.plateau-de-saclay.net/iso.wasm \
  | grep -iE 'content-encoding|content-type'   # -> content-encoding: br ; content-type: application/wasm
```
The wire size should be ~2.77 MB (vs ~3.68 MB on Pages).

## Notes

- Pre-compression tooling (`brotli`, `gzip`) lives only in the CI runner — the
  shipped wasm and the Go module are untouched, so the coverage gate is unaffected.
- The loader (`streamFetch` + progress bar) needs no change: the browser decodes
  br/gz transparently and the progress bar uses the stamped decompressed total.
