# frontcli — Front in your terminal

Fast, script-friendly CLI for [Front](https://frontapp.com). Manage conversations, messages,
contacts, tags, and more from the command line. JSON output, multiple accounts, and secure
credential storage built in.

## Features

- **Conversations** - list/search/get, messages/comments, archive/open/trash, assign/unassign, snooze, follow, custom fields
- **Messages** - get, send, reply, attachments + download
- **Drafts** - create, list, get, update, delete
- **Tags** - list/tree, get, create, update, delete, children, convos
- **Contacts** - list/search/get, handles, notes, convos, create/update/delete/merge
- **Inboxes** - list/get, convos, channels
- **Teammates** - list/get, convos
- **Channels** - list, get
- **Comments** - list/get/create (internal discussions)
- **Templates** - list/get/use (canned responses)
- **Whoami** - show authenticated user
- **Multiple accounts** - manage multiple Front accounts with aliases
- **Secure credential storage** using OS keyring (macOS Keychain, Linux Secret Service)
- **Auto-refreshing tokens** - authenticate once, use indefinitely
- **Parseable output** - JSON or TSV (`--plain`) mode for scripting and automation

## Installation

### Build from Source

```bash
git clone https://github.com/dedene/frontapp-cli.git
cd frontapp-cli
make build
```

Run:

```bash
./bin/frontcli --help
```

### Homebrew (coming soon)

```bash
brew install dedene/tap/frontcli
```

## Quick Start

### 1. Create a Front OAuth App

Before using frontcli, create an OAuth app in Front:

1. Go to **Settings → Developers** in Front: https://app.frontapp.com/settings/developers
2. Click **"New app"**
3. Configure the OAuth settings (see detailed guide below)
4. Save to generate your Client ID and Client Secret

### 2. Store OAuth Credentials

```bash
frontcli auth setup <client_id> <client_secret>
```

### 3. Authenticate

```bash
frontcli auth login
```

This opens a browser for OAuth authorization. On first use, you'll see a browser security warning
because frontcli uses a self-signed certificate for localhost. Click **"Advanced"** → **"Proceed to
localhost"** (Chrome) or similar in other browsers. This is a one-time step.

The refresh token is stored securely in your system keychain.

### 4. Test

```bash
frontcli conv list --limit 5
frontcli inbox list
frontcli teammate list
```

## Setting Up Your Front OAuth App

When creating a new app in Front's developer settings, configure these options:

### 1. Redirect URLs

Click **"Add Redirect URL"** and add:

```
https://localhost:8484/callback
```

This is the callback URL that frontcli uses during the OAuth flow. frontcli runs a local HTTPS
server with a self-signed certificate to receive the OAuth callback.

### 2. Namespace Access

Select which namespaces your app can access:

| Option                | Description                          |
| --------------------- | ------------------------------------ |
| **Global resources**  | Company-wide resources (recommended) |
| **Shared resources**  | Shared inboxes and conversations     |
| **Private resources** | Private inboxes (if needed)          |

For full access, check all three boxes.

### 3. Resource Permissions

Select permissions for each resource type. Here are the recommended permissions for full CLI
functionality:

| Resource              | Read | Write | Delete | Send |
| --------------------- | :--: | :---: | :----: | :--: |
| **Accounts**          |  ✓   |       |        |      |
| **Attachments**       |  ✓   |       |        |      |
| **Channels**          |  ✓   |       |        |      |
| **Comments**          |  ✓   |   ✓   |        |      |
| **Contacts**          |  ✓   |   ✓   |   ✓    |      |
| **Conversations**     |  ✓   |   ✓   |        |      |
| **Drafts**            |  ✓   |   ✓   |   ✓    |      |
| **Inboxes**           |  ✓   |       |        |      |
| **Message templates** |  ✓   |       |        |      |
| **Messages**          |  ✓   |   ✓   |        |  ✓   |
| **Tags**              |  ✓   |   ✓   |   ✓    |      |
| **Teammates**         |  ✓   |       |        |      |

**Tip:** Start with Read permissions only if you just want to query data. Add Write/Delete/Send as
needed.

### 4. Save and Get Credentials

1. Click **Save** at the bottom
2. Your **Client ID** and **Client Secret** will be generated
3. Copy these values for the `frontcli auth setup` command

## Authentication

### Storing Credentials

```bash
# Store OAuth credentials (required once)
frontcli auth setup <client_id> <client_secret>

# Authenticate with Front
frontcli auth login

# Check authentication status
frontcli auth status

# List authenticated accounts
frontcli auth list

# Log out
frontcli auth logout
```

### Multiple Accounts

Use the `--account` flag or `FRONT_ACCOUNT` environment variable:

```bash
# Via flag
frontcli conv list --account work@company.com

# Via environment
export FRONT_ACCOUNT=work@company.com
frontcli conv list
```

Override OAuth client selection with `--client`:

```bash
frontcli --client work-client conv list
```

### Keyring Backend

Tokens are stored securely using your system's keyring:

- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)

For environments without a keyring (CI, containers), use the file backend:

```bash
export FRONT_KEYRING_BACKEND=file
export FRONT_KEYRING_PASSWORD='your-password'
frontcli auth login
```

## Commands

### Conversations

```bash
# List conversations
frontcli conv list
frontcli conv list --inbox inb_xxx --limit 10
frontcli conv list --status open
frontcli conv list --tag tag_xxx

# Get conversation details
frontcli conv get cnv_xxx
frontcli conv get cnv_xxx --messages    # Include messages
frontcli conv messages cnv_xxx
frontcli conv comments cnv_xxx

# Search conversations
frontcli conv search "customer issue"
frontcli conv search --from client@co.com --tag tag_xxx --status open

# Manage conversation status
frontcli conv archive cnv_xxx cnv_yyy   # Archive multiple
frontcli conv archive --ids-from -      # Read IDs from stdin
frontcli conv open cnv_xxx              # Unarchive
frontcli conv trash cnv_xxx             # Move to trash

# Assign conversation
frontcli conv assign cnv_xxx --to tea_xxx
frontcli conv unassign cnv_xxx

# Snooze
frontcli conv snooze cnv_xxx --until "2024-01-15T09:00:00Z"
frontcli conv unsnooze cnv_xxx

# Followers
frontcli conv followers cnv_xxx
frontcli conv follow cnv_xxx
frontcli conv unfollow cnv_xxx

# Custom fields
frontcli conv update cnv_xxx --field "Priority=High" --field "Category=Support"

# Manage tags
frontcli conv tag cnv_xxx tag_xxx       # Add tag
frontcli conv untag cnv_xxx tag_xxx     # Remove tag
```

### Messages

```bash
# Get message
frontcli msg get msg_xxx
frontcli msg get msg_xxx --raw          # Show raw HTML body

# Send new message
frontcli msg send --channel cha_xxx --to user@example.com --subject "Hello" --body "Message body"
frontcli msg send --channel cha_xxx --to user@example.com --body-file ./message.txt

# Reply to conversation
frontcli msg reply cnv_xxx --body "Thanks for reaching out"
frontcli msg reply cnv_xxx --body-file ./reply.txt

# List attachments
frontcli msg attachments msg_xxx

# Download attachment
frontcli msg attachment download att_xxx -o ./file.pdf
```

### Drafts

```bash
# Create draft (reply to conversation)
frontcli draft create cnv_xxx --body "Draft reply"

# Create draft (new message via channel)
frontcli draft create --channel cha_xxx --to user@example.com --body "Draft message"

# List drafts in conversation
frontcli draft list cnv_xxx

# Get draft
frontcli draft get dra_xxx

# Update draft (optimistic locking with version)
frontcli draft update dra_xxx --body "Updated draft" --draft-version 1

# Delete draft
frontcli draft delete dra_xxx
```

### Tags

```bash
# List all tags
frontcli tag list
frontcli tag list --tree

# Get tag details
frontcli tag get tag_xxx

# Create tag
frontcli tag create --name "Urgent" --color red
frontcli tag create --name "Follow-up" --parent tag_xxx   # Child tag

# Update tag
frontcli tag update tag_xxx --name "Very Urgent"

# Delete tag
frontcli tag delete tag_xxx

# List child tags
frontcli tag children tag_xxx

# Conversations with tag
frontcli tag convos tag_xxx
```

### Contacts

```bash
# List contacts
frontcli contact list
frontcli contact list --limit 50
frontcli contact search "john"

# Get contact
frontcli contact get ctc_xxx
frontcli contact handles ctc_xxx

# Manage handles
frontcli contact handle add ctc_xxx --type email --value new@example.com
frontcli contact handle delete hdl_xxx

# Notes
frontcli contact notes ctc_xxx
frontcli contact note add ctc_xxx --body "Important customer"

# Conversations for contact
frontcli contact convos ctc_xxx

# Create contact
frontcli contact create --handle email:user@example.com --name "John Doe"
frontcli contact create --handle phone:+1234567890 --name "Jane Doe"

# Update contact
frontcli contact update ctc_xxx --name "John Smith"

# Delete contact
frontcli contact delete ctc_xxx

# Merge contacts
frontcli contact merge ctc_source ctc_target
```

### Other Resources

```bash
# Inboxes
frontcli inbox list
frontcli inbox get inb_xxx
frontcli inbox convos inb_xxx
frontcli inbox channels inb_xxx

# Teammates
frontcli teammate list
frontcli teammate get tea_xxx
frontcli teammate convos tea_xxx

# Channels
frontcli channel list
frontcli channel get cha_xxx

# Comments (internal discussions)
frontcli comment list cnv_xxx
frontcli comment get cmt_xxx
frontcli comment create cnv_xxx --body "Internal note"

# Templates
frontcli template list
frontcli template get rsp_xxx
frontcli template use rsp_xxx

# Whoami
frontcli whoami
```

## Output Formats

### Human-Readable (Default)

```bash
$ frontcli conv list --limit 3
ID              STATUS    ASSIGNEE           SUBJECT                    CREATED
cnv_abc123      open      alice@company.com  Re: Order question         2025-01-15 10:30
cnv_def456      archived  bob@company.com    Invoice inquiry            2025-01-14 15:45
cnv_ghi789      open      -                  New customer request       2025-01-14 09:20
```

### JSON (for scripting)

```bash
$ frontcli conv list --limit 1 --json
{
  "_results": [
    {
      "id": "cnv_abc123",
      "subject": "Re: Order question",
      "status": "open",
      ...
    }
  ]
}
```

Use JSON output with `jq` for powerful scripting:

```bash
# Get IDs of all open conversations
frontcli conv list --status open --json | jq -r '._results[].id'

# Archive all conversations with a specific tag
frontcli conv list --tag tag_xxx --json | jq -r '._results[].id' | xargs frontcli conv archive
```

### Plain (TSV)

```bash
$ frontcli conv list --limit 1 --plain
cnv_abc123	open	alice@company.com	Re: Order question	2025-01-15 10:30
```

## Configuration

### Environment Variables

| Variable                 | Description                                     |
| ------------------------ | ----------------------------------------------- |
| `FRONT_ACCOUNT`          | Default account email (avoids `--account` flag) |
| `FRONT_JSON`             | Set to `1` for JSON output by default           |
| `FRONT_PLAIN`            | Set to `1` for TSV output by default            |
| `FRONT_KEYRING_BACKEND`  | Keyring backend: `auto`, `keychain`, `file`     |
| `FRONT_KEYRING_PASSWORD` | Password for file-based keyring                 |

### Config File

Config is stored at:

- **macOS**: `~/Library/Application Support/frontcli/config.yaml`
- **Linux**: `~/.config/frontcli/config.yaml`

```yaml
default_account: work@company.com
account_aliases:
  work: work@company.com
  personal: me@gmail.com
default_output: text   # text | json | plain
timezone: UTC
```

### Config Commands

```bash
# Show config paths
frontcli config path
```

## Shell Completions

Generate completions for your shell:

```bash
# Bash
frontcli completion bash > /etc/bash_completion.d/frontcli
# Or: eval "$(frontcli completion bash)"

# Zsh
frontcli completion zsh > "${fpath[1]}/_frontcli"
# Or: eval "$(frontcli completion zsh)"

# Fish
frontcli completion fish > ~/.config/fish/completions/frontcli.fish
```

## Development

```bash
# Install tools
make tools

# Format code
make fmt

# Lint
make lint

# Test
make test

# Build
make build
```

## Security

- OAuth credentials are stored in `~/.config/frontcli/clients/` with 0600 permissions
- Refresh tokens are stored in your system's secure keyring
- Access tokens are kept in memory only and refreshed automatically
- Never commit credentials to version control

## Links

- [Front API Documentation](https://dev.frontapp.com/reference/introduction)
- [Front Developer Portal](https://app.frontapp.com/settings/developers)
- [GitHub Repository](https://github.com/dedene/frontapp-cli)

## License

MIT
