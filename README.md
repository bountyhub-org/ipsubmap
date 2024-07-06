# IPSubMap

IPSubMap is a tool that allows you to quickly scan a list of IP addresses and determine which subnets they belong to. This is useful for
mapping out hosts and identifying which subdomain points to it.

The tool can output 3 files:
- A list of private IP addresses
- A list of public IP addresses
- A list of loopback IP addresses

By default, it resolves both ipv4 and ipv6 addresses. You can turn off ipv6 resolution for example by using `-ipv6=false`.

The output format is simple, and is intended to be used by other utilities to transform it.

Output format:
```
<ip address> <domain>[,<domain>...]
```

First column is the IP address, and the second column is a comma separated list of domains that point to that IP address.

## Installation

You can install the tool by running:
```bash
go install github.com/bountyhub-org/ipsubmap@latest
```

## Usage examples

### Map subdomains.txt

In this example, list of subdomains is stored in `subdomains.txt`. The output files will be `loopback.txt`, `private.txt`, and `public.txt`.

```bash
ipsubmap -file subdomains.txt -out-loopback loopback.txt -out-private private.txt -out-public public.txt -ipv6=false
```

### Take out all public IP addresses

Now, let's say you want to list all IP addresses that are public:

```bash
cat public.txt | awk '{print $1}'
# Or you can use the cut command if you prefer
cat public.txt | cut -d ' ' -f1
```

### List all domains with public IP addresses

If you want to list all domains that point to public IP addresses:

```bash
cat public.txt | awk '{print $2}' | tr ',' '\n' | sort -u
# Or you can use the cut command if you prefer
cat public.txt | cut -d ' ' -f2 | tr ',' '\n' | sort -u
```
