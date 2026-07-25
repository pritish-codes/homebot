# Hosting HomeBot on CasaOS

This sets up: push to GitHub → GitHub Actions builds a Docker image → pushes it
to GHCR (private, same as your repo) → your CasaOS box polls for the new image
every few minutes via Watchtower and redeploys itself automatically → served
through your existing Cloudflare Tunnel.

No inbound ports need to be opened on your router — the CasaOS box only makes
outbound connections (to GHCR and to Cloudflare).

---

## 1. One-time GitHub setup (already done)

- `.github/workflows/publish-ghcr.yml` is already in the repo. Every push to
  `main` builds the image from the root `Dockerfile` and pushes it to
  `ghcr.io/pritish-codes/homebot:latest` (and `:<commit-sha>`).
- Because the repo is private, the resulting GHCR image is private too. Check
  this after your first push completes (Actions tab should show a green run):
  - Go to `https://github.com/pritish-codes/homebot/pkgs/container/homebot`
  - Confirm it exists and is marked **Private**

## 2. Create a GitHub Personal Access Token (PAT) for pulling images

The CasaOS box needs to authenticate to GHCR to pull a private image.

1. Go to GitHub → Settings → Developer settings → Personal access tokens →
   **Fine-grained tokens** → Generate new token
2. Scope: **Read-only access to Packages** (`read:packages`) — nothing else
3. Repository access: only `pritish-codes/homebot`
4. Copy the token somewhere safe (starts with `github_pat_...`) — you won't
   see it again

## 3. On the CasaOS terminal: authenticate Docker to GHCR

```bash
# replace <PAT> with the token from step 2, and <your-github-username>
echo '<PAT>' | docker login ghcr.io -u <your-github-username> --password-stdin
```

You should see `Login Succeeded`. This stores the credential in
`~/.docker/config.json` on the CasaOS host so Docker (and Watchtower) can
pull the private image going forward.

## 4. Create the data directory

```bash
mkdir -p /DATA/AppData/homebot
```

(Adjust the path to wherever CasaOS keeps app data on your system — check
`/DATA/AppData/` exists first with `ls /DATA/AppData/`.)

## 5. Create the compose file

```bash
mkdir -p ~/homebot
cat > ~/homebot/docker-compose.yml <<'EOF'
services:
  homebot:
    image: ghcr.io/pritish-codes/homebot:latest
    container_name: homebot
    restart: unless-stopped
    ports:
      - "3100:7745"
    environment:
      - TZ=America/New_York   # change to your timezone
      - HBOX_LOGGER_LEVEL=info
    volumes:
      - /DATA/AppData/homebot:/data
    labels:
      # scopes Watchtower to only this container
      - "com.centurylinklabs.watchtower.enable=true"

  watchtower:
    image: containrrr/watchtower
    container_name: watchtower
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ~/.docker/config.json:/config.json:ro
    environment:
      - WATCHTOWER_LABEL_ENABLE=true   # only touches labeled containers
      - WATCHTOWER_POLL_INTERVAL=300   # check every 5 minutes
      - WATCHTOWER_CLEANUP=true        # remove old images after update
    command: --interval 300
EOF
```

`WATCHTOWER_LABEL_ENABLE=true` + the label on the `homebot` service means
Watchtower **only** manages this one container — it will not touch any other
apps already running under CasaOS.

## 6. Start it

```bash
cd ~/homebot
docker compose up -d
docker compose logs -f homebot   # Ctrl+C once you see it come up cleanly
```

Visit `http://<casaos-ip>:3100` to confirm it's running before wiring up the
tunnel.

## 7. Cloudflare Tunnel ingress

You already have `cloudflared` running. Add a new public hostname pointing at
this container:

- If you manage the tunnel via the Cloudflare dashboard (Zero Trust → Access →
  Tunnels → your tunnel → Public Hostname):
  - **Subdomain**: `homebot` (or whatever you want)
  - **Domain**: your domain
  - **Service**: `HTTP` → `localhost:3100`
- If you manage it via a local `config.yml` on the CasaOS box instead, add:
  ```yaml
  ingress:
    - hostname: homebot.yourdomain.com
      service: http://localhost:3100
    # ... your existing rules stay below this
  ```
  then `sudo systemctl restart cloudflared` (or however you run it).

**Before making this public**: confirm password protection is enabled. Demo
mode (`UNSAFE_DISABLE_PASSWORD_PROJECTION`) must NOT be set in your compose
file — it isn't in the config above, just don't add it.

## 8. Verify the auto-deploy loop end-to-end

1. Make a trivial change locally (e.g. edit a label in
   `frontend/locales/en.json`), commit, push to `main`
2. Watch the GitHub Actions run finish (Actions tab, green check)
3. On the CasaOS box:
   ```bash
   docker logs watchtower --tail 50 -f
   ```
   Within 5 minutes you should see Watchtower find the new image, pull it,
   and recreate the `homebot` container
4. Refresh your public URL — the change should be live, and your inventory
   data should still be there (proves the volume mount is working)

---

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `docker login` fails | PAT scope wrong, or expired — regenerate with `read:packages` only |
| Watchtower never updates | Check `docker logs watchtower` for auth errors; re-run step 3 (config.json may be stale) |
| Container starts but data resets on every deploy | `/DATA/AppData/homebot` volume mount missing or wrong path — check with `docker inspect homebot \| grep -A3 Mounts` |
| Site works on LAN but not via tunnel | Ingress rule not added, or `cloudflared` needs a restart to pick up config changes |
| GitHub Action fails to push image | Repo Settings → Actions → General → Workflow permissions → confirm "Read and write permissions" is enabled for `GITHUB_TOKEN` |
