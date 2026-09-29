# hello

A gRPC service on the vikrant platform, serving
`hello.v1.HelloService`.

Write your code in [`internal/handlers/`](internal/handlers/). Everything
around it is already in place:

- a hardened server that protects itself from toxic clients;
- a drain that moves long-lived streams to other replicas on shutdown;
- a canonical client for your callers;
- a rootless image;
- a canary rollout;
- deny-by-default networking;
- autoscaling on in-flight RPCs;
- trace context propagated through every hop;
- one overlay per environment (dev, staging, prod);
- CI that proves all of it on every push, then publishes the image and
  pins dev to it.

## Who owns what

| Path | Owner | Changes when |
|---|---|---|
| `service.yaml`, `proto/` | you | you edit them |
| `internal/handlers/`, `cmd/`, `internal/platform/` | you, from day one | you edit them; the platform never writes here again |
| `go.mod`, `go.sum` | you | you add dependencies; the platform only raises a minimum version when its own code (the client, the probe) needs it |
| `client/` (canonical client and stubs), `deploy/base/`, `deploy/envs/*/kustomization.yaml`, `Dockerfile`, `Makefile`, `buf*.yaml`, `.github/`, `e2e/` | the platform | regenerated from `service.yaml` and the proto |
| `deploy/envs/*/image.yaml` | delivery | CI pins dev on every merge to `main`; the `promote` workflow moves a tested digest to the next environment |

Access is three lists in `service.yaml`, one per enforcement layer:
`authorizedCallers` (who may call which methods; Istio, by mTLS identity),
`ingress` (who may connect; a NetworkPolicy) and `egress` (what you may
connect to). Keep `ingress` in step with `authorizedCallers`; the platform
warns when they differ. To change any of them, edit `service.yaml` and
push. The platform regenerates the derived files and pushes a commit to the
same branch, so your PR shows the intent and its effect together. CI's
`check-stamp` fails if a platform-owned file is edited by hand.

Environments differ only in scale and rollout pace; security and the call
graph are the same everywhere. Override scale per environment in
`service.yaml` under `spec.environments`. Each environment's Argo CD syncs
`deploy/envs/<env>` from `main`.

## Run it

```sh
make run                  # the server on :50051, metrics on :9090
make test                 # unit tests + the toxic-client conformance suite
make tools check-generated  # stubs match the proto
make image                # the rootless production image
```

## Calling this service from another one

```go
import (
	hellov1 "github.com/smallStepGiantLeap/hello/client/gen/hello/v1"
	"github.com/smallStepGiantLeap/hello/client"
)

cc, err := client.Dial(client.Target)
c := hellov1.NewHelloServiceClient(cc)
```

To call another service, list it under `egress` with the methods you
need. If it has not authorized and admitted you, the platform opens an
access-request pull request on its repository; once that is merged, this
repository is re-rendered and its e2e expects the calls to succeed.
