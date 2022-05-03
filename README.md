# Info
Tool to mange IP notes and PTR in IBM Soflayer Classic Infrastructure
Creds must be provided with environment variables `SL_USER` and `SL_APIKEY`

# Build
go build

# Usage
./softlayer
  -force
        force yes to rename prompt. Use with caution!!!. default false
  -ip string
        ip address to delete in x.x.x.x form. default ''
  -note string
        note about ip in ibm cloud [hostname]. default 'FREE' (default "FREE")
  -ptr string
        ip address ptr [host.domain.zone]. default 'free' (default "none")
  -ttl int
        ttl for ptr. default 3600 (default 3600)

