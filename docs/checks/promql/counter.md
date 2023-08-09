---
layout: default
parent: Checks
grand_parent: Documentation
---

# promql/counter

This check inspects counter metrics used in queries to verify that the raw counter value is not used in calculations.

A counter should be wrapped by one of the following functions:

- `rate`, `irate`, `increase` - performing calculations while handling counter resets (https://promlabs.com/blog/2021/01/29/how-exactly-does-promql-calculate-rates/)
- `count`, `count_over_time`, `absent`, `absent_over_time` - these are common and valid use cases of any raw metric

## Common problems

### False Alarm

While this check is useful in identifying misuse of raw counters, we recognize there are rare but valid use cases.

E.g.
- a recording rule that evaluates a raw counter with label matchers (e.g. for filtering a subset of high cardinality metrics)
- an alert that uses a counter as an info metric to join labels

As a result, the check severity is set to `Warning` rather than `Bug`.

### Metadata mismatch

See [Same section in promql/rate check documentation](rate.md#metadata%20mismatch)

## Configuration

This check doesn't have any configuration options.

## How to enable it

This check is enabled by default for all configured Prometheus servers.

## How to disable it

You can disable this check globally by adding this config block:

```js
checks {
  disabled = ["promql/counter"]
}
```

You can also disable it for all rules inside given file by adding
a comment anywhere in that file. Example:

```yaml
# pint file/disable promql/counter
```

Or you can disable it per rule by adding a comment to it. Example:

```yaml
# pint disable promql/counter
```

If you want to disable only individual instances of this check
you can add a more specific comment.

```yaml
# pint disable promql/counter($prometheus)
```

Where `$prometheus` is the name of Prometheus server to disable.

Example:

```yaml
# pint disable promql/counter(prod)
```

## How to snooze it

You can disable this check until given time by adding a comment to it. Example:

```yaml
# pint snooze $TIMESTAMP promql/counter
```

Where `$TIMESTAMP` is either use [RFC3339](https://www.rfc-editor.org/rfc/rfc3339)
formatted  or `YYYY-MM-DD`.
Adding this comment will disable `promql/counter` *until* `$TIMESTAMP`, after that
check will be re-enabled.
