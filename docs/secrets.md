# Secrets

The single rule:

> **Secrets never live in `.env` files committed to the repo, and never reach
> the application via a flag or hard-coded default. They arrive through a
> platform-managed secret store, mounted into the container's environment at
> runtime.**

`.env` / `.env.example` are for **non-sensitive defaults** (ports, log level,
feature toggles). Anything that, if leaked, requires rotation goes elsewhere.

## The Go side — fail-fast on missing secrets

`pkg/config.MustGetSecret(key)` panics during boot if the env var is missing
or empty. Wire it for every credential:

```go
import "github.com/kris/go-infrastructure/pkg/config"

dsn := config.MustGetSecret("MYSQL_DSN")   // panics if absent
db, cleanup, err := data.NewMySQL(data.MySQLConfig{DSN: dsn}, logger)
```

If the deployment forgot to mount the secret, the pod **CrashLoopBackOffs
immediately with a clear message** — far better than the silent variant
where MYSQL_DSN is "" and every request authentication-fails.

`pkg/config.GetSecret(key)` is the non-panicking variant; use only when
the service can run usefully without that credential.

## The Kubernetes side — `envFrom: secretRef`

The chart in `kris-alpha/helm-charts/values.yaml` exposes an `envFrom` hook:

```yaml
envFrom:
  - secretRef:
      name: kris-alpha-credentials
```

Create the Secret separately (via `sealed-secrets`, `external-secrets`,
ArgoCD, Helm `--set`, etc. — **never with `kubectl create secret` ad-hoc
into prod**):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kris-alpha-credentials
type: Opaque
stringData:
  MYSQL_DSN: "..."
  REDIS_PASSWORD: "..."
```

Each key in the Secret becomes an environment variable on the pod.
`MustGetSecret("MYSQL_DSN")` reads it.

## Going further — pick one external store

For anything beyond a small team, **the Secret object should not be the
source of truth**. Pick an external store and one of the controllers that
sync into k8s:

| Approach                | Tool                                                              | When |
|-------------------------|-------------------------------------------------------------------|------|
| **Sealed Secrets**      | [`sealed-secrets`](https://github.com/bitnami-labs/sealed-secrets) | small to medium; encrypted manifests live in git, controller decrypts in-cluster |
| **External Secrets**    | [`external-secrets-operator`](https://external-secrets.io)        | sync from Vault / AWS Secrets Manager / GCP Secret Manager into k8s Secrets |
| **Vault Agent Injector**| [`vault-k8s`](https://github.com/hashicorp/vault-k8s)             | mature shops on Hashicorp Vault; mounts secrets as files on the pod |
| **CSI Secret Driver**   | [`secrets-store-csi-driver`](https://secrets-store-csi-driver.sigs.k8s.io) | cloud-native (AWS / Azure / GCP); secrets appear as mounted files |

**Pick exactly one per environment** and document it in your service's
deployment runbook. Mixing approaches multiplies failure modes.

## Logging hygiene

Even with all of the above, you can still leak a credential by logging it.
Mitigations baked into the scaffold:

- `pkg/middleware/access` only logs `kind/op/code/latency_ms/trace_id` — no
  bodies, no headers.
- `pkg/log.RedactHeaders(h, extra...)` masks `Authorization`, `Cookie`, etc.
  Use this any time you add custom request logging.

If you write your own logger calls that touch credential values, pass them
through `pkg/log.RedactValue(key, value)` first.

## What about `.env.global` and `.env.example`?

These are for **convenient local development**:

- `.env.example`     committed; documents what env vars the service reads
- `.env`             gitignored; your local overrides (Redis on a different port, etc.)
- `.env.global`      gitignored; project-root values shared across services in your local stack

`pkg/config.NewLoader(serviceName)` reads them in priority order: env vars
> service `.env` > global `.env.global` > `.env.example`.

**Production never sees a `.env`** — env vars come from k8s Secret /
ConfigMap directly. The loader's defaults are convenience for local dev,
not a deployment strategy.

## Rotation

Best practice every credential issuer endorses: **rotate on a schedule,
never wait for incident**. The scaffold doesn't bake in a rotation mechanism
— rotation is a property of the secret store, not the app. Make sure your
chosen approach above supports it:

- Sealed Secrets — re-encrypt + redeploy
- External Secrets — Vault / AWS rotation, ESO picks up changes
- Vault Agent — Vault leases; agent re-fetches before expiry

The pod **must** survive a rotation without manual intervention. If your
app caches credentials in memory at boot and never re-reads them, a rotation
breaks all replicas at once. `pkg/data.NewMySQL` reads the DSN once at boot
— this works because the `mysql` driver will reconnect with whatever the
current env var says **the next time the pod restarts**, and k8s schedules
the new Pod with the new Secret. For long-lived deploys spanning rotations,
you may need a credentials-reloading wrapper around the driver — out of
scope for this scaffold.
