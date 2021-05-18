# Contact-Auditor
A one-off tool produced to audit e-mail addresses present in the
contacts of subscriber IDs with unexpired certificates.

# Usage:

```shell
Usage of contact-auditor:
  -config string
        File containing a JSON config.
```

## Output:
Unless the audit encounters and error or an invalid e-mail address there
will be no output.

When an invalid e-mail address is encountered a line will be printed in
the following format:

```
validation failed for address: <e-mail> for ID: <ID> for reason: "<reason>"
```

# Configuration file:
The path to a database config file like the one below must be provided
following the `-config` flag.

```json
{
    "contactAuditor": {
      "passwordFile": "path/to/secretFile",
      "db": {
        "dbConnectFile": "path/to/secretFile",
        "maxOpenConns": 10
      }
    }
  }
  
```
