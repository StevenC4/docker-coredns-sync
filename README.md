# docker-coredns-sync

docker-coredns-sync listens for Docker container events and automatically registers or deregisters A and CNAME records in etcd using the [SkyDNS/CoreDNS etcd plugin format](https://coredns.io/plugins/etcd/). This enables dynamic DNS resolution for containers via CoreDNS.

---

## Features
* Supports **A** and **CNAME** records
* Multiple domain support per container
* Prevents CNAME cycles
* Automatically removes stale records
* Graceful shutdown support
* Label-based configuration

---

## Docker Label Configuration

Add these labels to your Docker containers to register DNS records:

### Required
* `coredns.enabled=true` — Enables DNS registration for this container.

### A Record

* `coredns.a.name=hoarder.carroll.live`
* `coredns.a.value=192.168.200.1` -  Optional (defaults to HOST_IP)

### CNAME Record

* `coredns.cname.name=alias.carroll.live`
* `coredns.cname.value=hoarder.carroll.live`

### Multiple Records

You can define multiple A or CNAME records by appending numeric indices:

* `coredns.a.1.name=api.carroll.live`
* `coredns.a.1.value=192.168.200.2`
* `coredns.cname.2.name=www.carroll.live`
* `coredns.cname.2.value=api.carroll.live`

### Optional
* `coredns.force=true` — Forces registration even if the record already exists and is owned by this host.

---

## Environment Variables

Set these to configure sync behavior:

| Variable | Description | Default |
| --- | --- | --- |
| `HOST_IP` | Default IP for A records | `127.0.0.1` |
| `HOSTNAME` | Unique host identifier | *your-hostname* |
| `ETCD_HOST` | etcd host | `localhost` |
| `ETCD_PORT` | etcd port | `2379` |
| `ETCD_PATH_PREFIX` | etcd base key path for DNS records | `/skydns` |
| `DOCKER_LABEL_PREFIX` | Label namespace to use (e.g. `coredns`) | `coredns` |
| `LOG_LEVEL` | Logging level (**DEBUG**, **INFO**, etc.) | **INFO** |
---

## etcd Record Format

Records are stored in etcd using SkyDNS naming format:
```
/skydns/<reversed.domain.name>/xN
```

### Example A Record

Key:
```
/skydns/live/carroll/hoarder/x1
```

Value:
```json
{
  "record_type": "A",
  "host": "192.168.200.1",
  "owner": "mozart"
}
```

### Example CNAME Record

Key:
```
/skydns/live/carroll/alias/x1
```

Value:
```json
{
  "record_type": "CNAME",
  "host": "hoarder.carroll.live",
  "owner": "mozart"
}
```

CNAME cycles are automatically detected and prevented.

---

## CoreDNS Setup

Example CoreDNS configuration for dynamic resolution via etcd:

```hcl
.:5335 {
    etcd carroll.live {
        path /skydns
        endpoint http://127.0.0.1:2379
        fallthrough
    }

    forward . 1.1.1.1 1.0.0.1
    cache 30
    log
}
```

You can use Pi-hole or your local system to point queries to this port.

---

## Running

Install dependencies and run:

```bash
pip install -r requirements.txt
python3 -m src.main
```

The sync script:
* Performs an initial sync on startup
* Watches live container events (`start`, `die`, `destroy`)
* Automatically adds/removes records in etcd
