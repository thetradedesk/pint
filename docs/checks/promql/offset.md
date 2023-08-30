---
layout: default
parent: Checks
grand_parent: Documentation
---

# promql/offset

This check inspects metric selectors whose name matches a specified prefix and doesn't have a corresponding minimum offset.
Some cloud monitoring services (e.g. AWS CloudWatch) metrics' have a delay before reaching Prometheus,
so queries on these metrics should be used with an offset that matches or exceeds the delay.

## Configuration

Syntax:

```js
offset {
  prefix   = "$pattern"  
  min      = "###u"
  severity = "severity"
}
```

- `prefix`   - regexp pattern to match metric name prefix on,
- `min`      - minimum duration the look back offset must be set to, `10m` would mean 10 minutes
- `severity` - optional severity level when check fails. Defaults to `warning` if not specified.

## How to enable it

This check is not enabled by default as it requires explicit configuration
to work.
To enable it add one or more `rule {...}` blocks and specify all required
rules there.

Examples:

Ensure that all metrics beginning with `aws_` have a minimum offset of 5 minutes:

```js
rule {
  offset {
    prefix   = "aws_.*"
    min      = "5m"
    severity = "bug"
  }
}
```

Handle multiple patterns with different offsets:

```js
rule {
  offset {
    prefix   = "pattern1_.*"
    min      = "5m"
    severity = "bug"
  }
  
  offset {
    prefix   = "(pattern2_|pattern3).*"
    min      = "7m"
  }
}
```

## How to disable it

You can disable this check globally by adding this config block:

```js
checks {
  disabled = ["promql/offset"]
}
```

You can also disable it for all rules inside given file by adding
a comment anywhere in that file. Example:

```yaml
# pint file/disable promql/offset
```

Or you can disable it per rule by adding a comment to it. Example:

```yaml
# pint disable promql/offset
```

## How to snooze it

You can disable this check until given time by adding a comment to it. Example:

```yaml
# pint snooze $TIMESTAMP promql/offset
```

Where `$TIMESTAMP` is either use [RFC3339](https://www.rfc-editor.org/rfc/rfc3339)
formatted  or `YYYY-MM-DD`.
Adding this comment will disable `promql/offset` *until* `$TIMESTAMP`, after that
check will be re-enabled.
