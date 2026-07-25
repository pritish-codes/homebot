<div align="center">
  <img src="/docs/src/assets/lilbox.svg" height="200"/>
</div>

<h1 align="center" style="margin-top: -10px;"> HomeBot </h1>
<p align="center" style="width: 100%;">
   A personal fork of <a href="https://github.com/sysadminsmedia/homebox">Homebox</a>, self-hosted on a home server.
</p>

## What is HomeBot

HomeBot is a home inventory and organization system, forked from Homebox with an added focus on **recurring maintenance scheduling** — track not just what you own, but when it's due for upkeep. While extending this project, we've tried to keep Homebox's original principles in mind:

- 🧘 _Simple but Expandable_ - designed to be simple and easy to use, but expandable to whatever level of infrastructure you want to put into it.
- 🚀 _Blazingly Fast_ - written in Go, extremely fast and requires minimal resources to deploy. Idle memory usage is generally under 50MB for the whole container.
- 📦 _Portable_ - uses SQLite and an embedded Web UI to make it easy to deploy, use, and back up.

### Key Features
- 📇 Rich Organization - Organize your items into categories, locations, and tags. Create custom fields to store additional information about your items.
- 🔍 Powerful Search - Quickly find items in your inventory.
- 📸 Image Upload - Upload images of your items to make it easy to identify them.
- 📄 Document and Warranty Tracking - Keep track of important documents and warranties for your items.
- 💰 Purchase & Recurring Maintenance Tracking - Track purchase dates, prices, and recurring maintenance schedules (e.g. "replace HVAC filter every 3 months") for your items.
- 📱 Responsive Design - Works on any device, including desktops, tablets, and smartphones.

## Screenshots
![Login Screen](.github/screenshots/1.png)
![Dashboard](.github/screenshots/2.png)
![Item View](.github/screenshots/3.png)
![Create Item](.github/screenshots/9.png)
![Search](.github/screenshots/8.png)

## Quick Start

```bash
# If using the rootless or hardened image, ensure data
# folder has correct permissions
mkdir -p /path/to/data/folder
chown 65532:65532 -R /path/to/data/folder
docker run -d \
  --name homebot \
  --restart unless-stopped \
  --publish 3100:7745 \
  --env TZ=Europe/Bucharest \
  --volume /path/to/data/folder/:/data \
  ghcr.io/pritish-codes/homebot:latest
```

## Credits

HomeBot is a fork of [Homebox](https://github.com/sysadminsmedia/homebox), used and modified here under the terms of the [GNU AGPLv3 license](./LICENSE).

- Original Homebox project by [@hay-kot](https://github.com/hay-kot)
- Currently maintained upstream by [sysadminsmedia](https://github.com/sysadminsmedia)
- Original logo by [@lakotelman](https://github.com/lakotelman)

If you're looking for the actively maintained open-source community project (Discord, translations, hosted demo, etc.), see the upstream [Homebox repository](https://github.com/sysadminsmedia/homebox).
