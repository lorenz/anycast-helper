# Anycast Helper
*Small helper adding and removing Anycast IPs from interfaces depending on listeners*

## Usage
```
anycast-helper --anycast-ip 192.168.32.1 --port 1234
```
* Requires at least CAP_NET_ADMIN beacause it needs write access to netlink
* Uses (and creates) the `anycast0` interface for binding IPs to, this interface should be monitored by the routing daemon
* Sync period is fixed to 500ms which should be plenty fast while using extremely little resources