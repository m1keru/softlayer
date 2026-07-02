# softlayer

Author: **m1keru** · [1@m1ke.ru](mailto:1@m1ke.ru)

CLI tool to manage IP address notes and PTR records in IBM SoftLayer
(Classic Infrastructure).

Credentials must be provided with the `SL_USER` and `SL_APIKEY` environment
variables.

## Build

```bash
make build       # binary for the host platform
make build-all   # static binaries for linux/darwin x amd64/arm64 into dist/
make install     # install to /usr/local/bin (replaces the old binary)
```

If `which softlayer` points to an older build, run `make install` or call
the binary from this repo directly (`./softlayer`).

## Usage

```text
softlayer <command> [options]

Commands:
  list        list subnets and their IPs (free ones by default)
  set         set PTR and/or note for an IP
  clear       remove PTR and note from an IP
  lease       reserve the first free private+public IP pair and print
              puppet/netplan network config
  stale       list public IPs that do not answer pings
  completion  print shell completion script (bash or zsh)
  version     print the version number
```

### list

Lists free IPs (no PTR and no note) grouped by subnet. By default subnets
whose IBM note contains `ONLY FOR METAL`, `FULL` or `NETWORK FOR INFOSEC`
are excluded. Use `-exclude-note` / `-exclude-cidr` to add more exclusions,
or `-no-default-excludes` to search all subnets.

```bash
softlayer list                 # both public and private subnets
softlayer list -public         # only public subnets
softlayer list -private        # only private subnets
softlayer list -all            # show every IP with its PTR and note
softlayer list -public -json   # machine-readable output
softlayer list -private -one   # first free IP as 'ip/mask gateway'
softlayer list -exclude-cidr 10.66.0.0/24,203.0.113.0/24
softlayer list -exclude-note "LAB NETWORK" -no-default-excludes
```

### set

Sets the PTR record and/or the note of an IP. Only the flags you pass are
applied. PTR updates are skipped for private (RFC 1918) addresses since they
have no public reverse zone. Passing an empty value deletes the PTR / clears
the note.

```bash
softlayer set -ip 198.51.100.10 -ptr my.example.com -note my.example.com -ttl 3600
softlayer set -ip 198.51.100.10 -note "reserved for lb"   # note only
softlayer set -ip 198.51.100.10 -ptr ""                   # delete PTR only
```

### clear

Removes the PTR record and clears the note in one go.

```bash
softlayer clear -ip 198.51.100.10
```

### lease

Finds the first free private and public IPs, assigns the PTR (public IP
only) and the note to both, then prints ready-to-use puppet and netplan
network configuration for the pair.

```bash
softlayer lease -ptr app.example.com -note app.example.com -force
softlayer lease -ptr host.example.com -note host.example.com \
  -exclude-cidr 10.66.1.0/24
```

The DNS search domain in the generated config defaults to `example.com`
and can be changed with `-search`.

### stale

Pings every public IP and prints the ones that do not answer — candidates
for reclaiming. By default excludes subnets noted `ONLY FOR METAL` and
`NETWORK FOR INFOSEC` (but not `FULL`). Same `-exclude-note`, `-exclude-cidr`
and `-no-default-excludes` flags as `list`.

```bash
softlayer stale
softlayer stale -json
softlayer stale -exclude-note FULL
```

`set`, `clear` and `lease` ask for confirmation before changing anything;
pass `-force` to skip the prompt (use with caution).

## Shell completions

Completion scripts for bash and zsh are embedded into the binary
(sources live in `completions/`).

```bash
# bash: current session
source <(softlayer completion bash)
# bash: permanently
softlayer completion bash > /etc/bash_completion.d/softlayer

# zsh: put the script into any directory from $fpath
softlayer completion zsh > "${fpath[1]}/_softlayer"
```

## Development

```bash
make test    # unit tests with the race detector
make lint    # golangci-lint (see .golangci.yml)
make clean   # remove build artifacts
```

CI (GitHub Actions) runs tests and golangci-lint on every push and pull
request, then cross-compiles binaries for all supported platforms and uploads
them as workflow artifacts.

## Releases

Push a semver tag to build and publish release binaries (linux/darwin,
amd64/arm64):

```bash
git tag v1.0.0
git push origin v1.0.0
```

The [Release workflow](.github/workflows/release.yml) attaches these files
to the GitHub Release:

- `softlayer-linux-amd64`
- `softlayer-linux-arm64`
- `softlayer-darwin-amd64`
- `softlayer-darwin-arm64`
- `checksums.txt` (sha256)
