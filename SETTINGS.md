# Settings

- Organization, access and secret key's: Scaleway credentials at https://console.scaleway.com/account/credentials
- Menu format: Template using for systray menu.
- Copy format: Template using for on-click copy.
- Check interval: Interval for getting data from Scaleway, in sec. Set less what 10 for disabling.
- Ping interval: Interval for servers ping, in sec. Set 0 for disabling.

## Templates

Templates use special `{KEY}` format for replacement on server data

- ID: Server id.
- NAME: Server name.
- IPv4: Public IPv4.
- IPv6: Public IPv6.
- IPvX: Public IPv4 or IPv6.
- STATE: Server status.
- REGION: Server region.
- PING: Ping to server in ms.

**Only for Menu format**:

- FLAG: Country flag from region, 🇫🇷 or 🇳🇱.
- ALIVE: Ping status, ✅ or ❌.
