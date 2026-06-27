# GitHub Actions Guide

This project has two workflows:

```txt
.github/workflows/ci.yml
.github/workflows/deploy.yml
```

## CI Workflow

Runs on:

```txt
push to master
pull request
```

It does:

```txt
1. Checkout code
2. Setup Go
3. Run tests
4. Build the app
5. Build Docker image
```

## Deploy Workflow

Runs on:

```txt
new production tag matching Vx.x.x from master
manual trigger with a production tag
```

Production tags must use this format:

```txt
V1.2.3
```

The tagged commit must exist on the `master` branch. Tags created from another branch will fail before deployment.

It SSHs into the VPS and runs:

```bash
cd /opt/cermin-backend
git fetch --tags --force origin
git checkout --force V1.2.3
docker compose up -d --build
docker image prune -f
```

## Required GitHub Secrets

Open:

```txt
GitHub repository
Settings
Secrets and variables
Actions
New repository secret
```

Add:

```txt
VPS_HOST=your-server-ip-or-domain
VPS_USER=root
VPS_SSH_PORT=22
VPS_APP_DIR=/opt/cermin-backend
VPS_SSH_KEY=your-private-ssh-key
```

`VPS_SSH_KEY` must be the private key that can SSH into your server.

The matching public key must exist on the server in:

```txt
~/.ssh/authorized_keys
```

## Manual Deploy

Open:

```txt
GitHub repository
Actions
Deploy Production
Run workflow
```

Enter the production tag you want to deploy, for example:

```txt
V1.2.3
```

## Production Deploy

Create and push a production tag:

```bash
git checkout master
git pull origin master
git tag V1.2.3
git push origin V1.2.3
```

GitHub Actions will validate the tag, confirm the tagged commit is on `master`, run tests, build the app, build the Docker image, and deploy the exact tagged commit to the VPS.

## Important

The VPS must already have:

```txt
Docker
Docker Compose
Project cloned in /opt/cermin-backend
.env file created on the server
```

Do not commit `.env` to GitHub.
