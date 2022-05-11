# Info
Tool to mange IP notes and PTR in IBM Soflayer Classic Infrastructure.

__Creds must be provided with environment variables `SL_USER` and `SL_APIKEY`__

# Build
go build

# Usage

```bash
./softlayer
  -force
        force yes to rename prompt. Use with caution!!!. default false
  -ip string
        ip address to delete in x.x.x.x form. default ''
  -list
        list free public and private ips
  -note string
        note about cli in ibm cloud [host.domain.com]. default ''
  -private
        list only free private ips
  -ptr string
        cli address ptr [hostname]. default ''
  -public
        list only free public ips
  -ttl int
        ttl for ptr. default 3600 (default 3600)
```
