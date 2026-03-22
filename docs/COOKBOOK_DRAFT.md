# OTTL Cookbook — DRAFT

> **Status:** draft — created during GSoC 2026 community bonding, not final.

## 1. GKE Log Normalization (Issue #30800)

This is the single most-requested feature from the community and the direct motivating case for the entire for-range proposal.

### The Problem

When running on GKE, the resource detector populates resource-level attributes like `cloud.provider`, `k8s.cluster.name`, `k8s.pod.name`, and ~20 others. A common requirement is to copy all of these into log record attributes so downstream systems (Loki, BigQuery, Splunk) can query them without joining on resource metadata. Today this requires one `set()` call per attribute:

```yaml
transform:
  log_statements:
    - context: log
      statements:
        - set(attributes["cloud.provider"], resource.attributes["cloud.provider"]) where resource.attributes["cloud.provider"] != nil
        - set(attributes["cloud.platform"], resource.attributes["cloud.platform"]) where resource.attributes["cloud.platform"] != nil
        - set(attributes["cloud.account.id"], resource.attributes["cloud.account.id"]) where resource.attributes["cloud.account.id"] != nil
        - set(attributes["cloud.region"], resource.attributes["cloud.region"]) where resource.attributes["cloud.region"] != nil
        - set(attributes["cloud.availability_zone"], resource.attributes["cloud.availability_zone"]) where resource.attributes["cloud.availability_zone"] != nil
        - set(attributes["k8s.cluster.name"], resource.attributes["k8s.cluster.name"]) where resource.attributes["k8s.cluster.name"] != nil
        - set(attributes["k8s.namespace.name"], resource.attributes["k8s.namespace.name"]) where resource.attributes["k8s.namespace.name"] != nil
        - set(attributes["k8s.pod.name"], resource.attributes["k8s.pod.name"]) where resource.attributes["k8s.pod.name"] != nil
        - set(attributes["k8s.pod.uid"], resource.attributes["k8s.pod.uid"]) where resource.attributes["k8s.pod.uid"] != nil
        - set(attributes["k8s.container.name"], resource.attributes["k8s.container.name"]) where resource.attributes["k8s.container.name"] != nil
        - set(attributes["k8s.node.name"], resource.attributes["k8s.node.name"]) where resource.attributes["k8s.node.name"] != nil
        - set(attributes["k8s.deployment.name"], resource.attributes["k8s.deployment.name"]) where resource.attributes["k8s.deployment.name"] != nil
        - set(attributes["k8s.replicaset.name"], resource.attributes["k8s.replicaset.name"]) where resource.attributes["k8s.replicaset.name"] != nil
        - set(attributes["k8s.statefulset.name"], resource.attributes["k8s.statefulset.name"]) where resource.attributes["k8s.statefulset.name"] != nil
        - set(attributes["k8s.daemonset.name"], resource.attributes["k8s.daemonset.name"]) where resource.attributes["k8s.daemonset.name"] != nil
        - set(attributes["k8s.job.name"], resource.attributes["k8s.job.name"]) where resource.attributes["k8s.job.name"] != nil
        - set(attributes["k8s.cronjob.name"], resource.attributes["k8s.cronjob.name"]) where resource.attributes["k8s.cronjob.name"] != nil
        - set(attributes["host.name"], resource.attributes["host.name"]) where resource.attributes["host.name"] != nil
        - set(attributes["host.id"], resource.attributes["host.id"]) where resource.attributes["host.id"] != nil
        - set(attributes["host.type"], resource.attributes["host.type"]) where resource.attributes["host.type"] != nil
        - set(attributes["os.type"], resource.attributes["os.type"]) where resource.attributes["os.type"] != nil
        - set(attributes["service.name"], resource.attributes["service.name"]) where resource.attributes["service.name"] != nil
        - set(attributes["service.namespace"], resource.attributes["service.namespace"]) where resource.attributes["service.namespace"] != nil
        - set(attributes["service.version"], resource.attributes["service.version"]) where resource.attributes["service.version"] != nil
```

That's 24 nearly-identical lines. If a new semantic convention attribute is added, someone has to remember to add another line. If they forget, that attribute silently disappears.

### The Solution — for-range

```yaml
transform:
  log_statements:
    - context: log
      statements:
        - |
          for key, val in resource.attributes {
            # Copy every resource attribute to log attributes.
            # The loop variable `key` is a string, `val` is the pcommon.Value.
            # Both are bound in loopScope per iteration and resolved during
            # path expression evaluation before falling back to pdata accessors.
            set(attributes[key], val)
          }
```

Five lines replace twenty-four. New attributes are automatically included. No maintenance burden.

### How It Works Internally

1. `resource.attributes` resolves to a `pcommon.Map` via the existing `PathGetSetter` in the `ottllog` context package.
2. The for-range evaluator snapshots the map's keys into a `[]string` before entering the loop. This prevents concurrent-modification panics if the body were to add or delete keys.
3. `loopScope` is allocated once: `make(map[string]pcommon.Value, 2)`.
4. Per iteration, `clear(loopScope)` resets without deallocating, then `key` and `val` are bound.
5. `set(attributes[key], val)` executes using normal OTTL semantics — `key` resolves from `loopScope` because `GetLoopVar("key")` is checked before the standard pdata accessor chain.
6. After the loop, `loopScope` is set to nil — subsequent non-loop statements pay zero cost.

**Performance:** The key snapshot adds O(n) overhead at loop entry. Each iteration runs at the same cost as a standalone `set()` call. For a typical GKE resource with ~24 attributes, total overhead is ≤2× vs unrolled.

---

## 2. PII Redaction with IsMatch

### The Problem

Logs from authentication services may contain PII in arbitrary attribute values — email addresses, IP addresses, potential credit card numbers. Operators need to scan all attributes and redact matching values without knowing the exact key names at config time.

### The Solution

```yaml
transform:
  log_statements:
    - context: log
      statements:
        - |
          for key, val in attributes
            where IsMatch(val, "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$")
               or IsMatch(val, "^\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}$")
               or IsMatch(val, "^\\d{13,16}$")
          {
            set(attributes[key], "***REDACTED***")
          }
```

The `where` guard on the for-range evaluates per iteration. The `or` combinator short-circuits: if the value matches an email pattern, it doesn't bother checking for IP or credit card patterns. Only matching values are replaced.

---

## 3. Metric Label Normalization

Normalize metric label keys from `camelCase` to `snake_case` across all data points in a metric stream. This serves platform engineers who receive metrics from services with inconsistent naming conventions and need a uniform schema for dashboarding.

## 4. Span Attribute Enrichment

Copy selected span attributes to resource-level for backends that do not support span-level attribute querying. This serves observability engineers who use backends like BigQuery that flatten resources and need all queryable fields at the same level.

## 5. Log Body Structured Extraction

Parse a JSON log body and distribute its fields into typed attributes using `ParseJSON()` inside a for-range loop. This serves SREs who receive unstructured JSON logs and need to query on individual fields without changing the log producer.

## 6. Dynamic Attribute Filtering

Remove all attributes matching a glob pattern using `IsMatch()` and `delete_key()` inside a for-range loop. This serves compliance teams who need to strip entire categories of attributes (e.g. all `debug.*` keys) before forwarding to a regulated backend.

## 7. Multi-Signal Correlation

Copy trace context fields (`trace_id`, `span_id`) from spans to corresponding log records using for-range over a joined context. This serves distributed systems engineers who need to correlate logs and traces in systems that don't natively propagate W3C trace context.

## 8. Compliance Tagging

Scan all telemetry attributes for compliance-relevant field patterns (PCI DSS card numbers, HIPAA identifiers) and add compliance classification tags automatically. This serves security engineers who need to ensure every signal carrying sensitive data is tagged for audit routing.
