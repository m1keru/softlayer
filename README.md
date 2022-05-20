# Info
Tool to mange IP notes and PTR in IBM Soflayer Classic Infrastructure.

__Creds must be provided with environment variables `SL_USER` and `SL_APIKEY`__

# Build
go build

# Usage

```bash
./softlayer
  -force
    	force yes to rename prompt. Use with caution!!! default false
  -ip string
    	ip address to deal with in x.x.x.x form. default ''
  -list
    	list free public and private ips
  -note string
    	note about cli in ibm cloud [host.domain.com]. default ''
  -private
    	list only free private ips [use only with -list]
  -ptr string
    	cli address ptr [hostname]. default ''
  -public
    	list only free public ips [use only with -list]
  -ttl int
    	ttl for ptr. default 3600
```

# Example
## list
```
softlayer -list -public 
```
## create 
```
softlayer -ip 10.114.97.209 -ptr="tor1-prod-apiconfig-6.etrigan.net" -note "tor1-prod-apiconfig-6.etrigan.net" -ttl=3600
```
## delete
```
softlayer -ip 10.114.97.209 -ptr= -note= 

or just

softlayer --ip=10.114.97.209
```
