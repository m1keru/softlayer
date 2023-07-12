# Info
Tool to mange IP notes and PTR in IBM Soflayer Classic Infrastructure.

__Creds must be provided with environment variables `SL_USER` and `SL_APIKEY`__

# Build
go build

# Usage

```bash
Usage of ./softlayer:
  -debug
        set logger to debug
  -force
        force yes to rename prompt. Use with caution!!! default false
  -ip string
        ip address to update in x.x.x.x form. default ''
  -lease
        lease ip [ptr,note - required]
  -list
        list free public and private ips
  -note string
        note about cli in ibm cloud [host.domain.com]. default ''
  -one
        get first free ip [use only with -list]
  -private
        list only free private ips [use only with -list]
  -ptr string
        cli address ptr [hostname]. default ''
  -public
        list only free public ips [use only with -list]
  -ttl int
        ttl for ptr. (default 3600)
```

# Example
## list
```
softlayer -list -public
```
## create
```
softlayer -ip 1.2.3.4 -ptr="test.example.net" -note "test.example.net" -ttl=3600
```
## delete
```
softlayer -ip 1.2.3.4 -ptr= -note=

or just

softlayer --ip=1.2.3.4
```

## lease
```
./softlayer --lease --ptr="test.example.net" --note="test.example.net"
```

## Get one ip rather then list
```
./softlayer --list --one
```
