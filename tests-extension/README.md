# cluster-dns-operator OTE Test Extension

All 7 DNS operator tests have been migrated from openshift-tests-private to the OTE (openshift-tests-extension) framework.

## Test Suites

### cluster-dns-operator/all
All 7 tests (use `--max-concurrency=1` since most are disruptive)

### cluster-dns-operator/non-disruptive
1 non-disruptive test (safe for development clusters):
- 26151 - Integrate DNS operator metrics with Prometheus

### cluster-dns-operator/disruptive
6 disruptive tests (may affect cluster state, all serial):
- 37912 - DNS operator should show clear error message when DNS service IP already allocated
- 41049 - DNS controls pod placement by node selector
- 41050 - DNS controls pod placement by tolerations
- 46183 - DNS operator supports Random, RoundRobin and Sequential policy for servers.forwardPlugin
- 46873 - Configure operatorLogLevel under the default dns operator and check the logs flag
- 63512 - Enabling force_tcp for protocolStrategy field to allow DNS queries to send on TCP to upstream server

### cluster-dns-operator/conformance/parallel
Level0 non-serial, non-disruptive tests

### cluster-dns-operator/conformance/serial
Level0 serial, non-disruptive tests

## How to Run

```bash
# Run these commands from the tests-extension directory
cd tests-extension

export KUBECONFIG=/path/to/kubeconfig

# Build the binary
make build

# List all available suites
./bin/cluster-dns-operator-tests-ext list suites

# List all tests
./bin/cluster-dns-operator-tests-ext list tests

# Run a single test by full name
./bin/cluster-dns-operator-tests-ext run-test "[OTP][sig-network-edge] Network_Edge Component_DNS Author:mjoseph-Critical-41050-DNS controller pod placement by tolerations [Disruptive] [Serial]"

# Run multiple specific tests (repeat -n/--names flag)
./bin/cluster-dns-operator-tests-ext run-test --names \
  "[OTP][sig-network-edge] Network_Edge Component_DNS Author:mjoseph-Critical-41049-DNS controls pod placement by node selector [Disruptive] [Serial]" \
  "[OTP][sig-network-edge] Network_Edge Component_DNS Author:mjoseph-Critical-41050-DNS controller pod placement by tolerations [Disruptive] [Serial]"

# Run all tests (serially recommended)
./bin/cluster-dns-operator-tests-ext run-suite cluster-dns-operator/all --max-concurrency=1

# Run only non-disruptive tests
./bin/cluster-dns-operator-tests-ext run-suite cluster-dns-operator/non-disruptive

# Run only disruptive tests (serially)
./bin/cluster-dns-operator-tests-ext run-suite cluster-dns-operator/disruptive --max-concurrency=1
```

**Important**: 6 of 7 tests are `[Disruptive] [Serial]` and **must** run with `--max-concurrency=1`. The OTE `run-suite` command does not parse `[Serial]` from test names to enforce serial execution — it defaults to `--max-concurrency=10`, which will run all tests in parallel. These tests all modify the shared `dns.operator.openshift.io/default` resource and call `deleteDnsOperatorToRestore` in defers, so running them concurrently causes cascading failures:

- Tests delete `dns.operator.openshift.io/default` while other tests are still using it (`NotFound` errors)
- DNS CO stays degraded because multiple tests modify DNS state simultaneously
- Node selector and toleration patches get reset by other tests' cleanup routines

Always use `--max-concurrency=1` for any suite that includes disruptive tests.

## Test Execution Time

- Single test: 2-10 minutes
- Non-disruptive suite: 5-10 minutes
- Disruptive suite: 15-30 minutes
- All tests: 20-40 minutes

## All Test Names

For copy-paste use with `run-test`:

```text
[OTP][sig-network-edge] Network_Edge Component_DNS Author:mjoseph-Critical-26151-Integrate DNS operator metrics with Prometheus [Skipped:MicroShift]
[OTP][sig-network-edge] Network_Edge Component_DNS Author:mjoseph-NonHyperShiftHOST-High-37912-DNS operator should show clear error message when DNS service IP already allocated [Disruptive] [Serial] [Skipped:MicroShift]
[OTP][sig-network-edge] Network_Edge Component_DNS Author:mjoseph-Critical-41049-DNS controlls pod placement by node selector [Disruptive] [Serial]
[OTP][sig-network-edge] Network_Edge Component_DNS Author:mjoseph-Critical-41050-DNS controll pod placement by tolerations [Disruptive] [Serial]
[OTP][sig-network-edge] Network_Edge Component_DNS Author:hongli-High-46183-DNS operator supports Random, RoundRobin and Sequential policy for servers.forwardPlugin [Disruptive] [Serial]
[OTP][sig-network-edge] Network_Edge Component_DNS Author:shudili-NonHyperShiftHOST-Medium-46873-Configure operatorLogLevel under the default dns operator and check the logs flag [Disruptive] [Serial]
[OTP][sig-network-edge] Network_Edge Component_DNS Author:mjoseph-High-63512-Enbaling force_tcp for protocolStrategy field to allow DNS queries to send on TCP to upstream server [Disruptive] [Serial]
```

## Files

```text
tests-extension/
├── cmd/main.go                    # OTE entry point, suite definitions
├── test/e2e/
│   ├── dns-operator.go            # 7 test implementations
│   ├── util.go                    # Helper functions (getByJsonPath, getPodListByLabel, etc.)
│   ├── bindata.mk                 # Bindata generation makefile
│   └── testdata/
│       ├── fixtures.go            # FixturePath helper
│       └── bindata.go             # Embedded test fixtures
├── Makefile                       # Build targets
├── go.mod                         # Dependencies
├── go.sum                         # Dependency checksums
├── vendor/                        # Vendored dependencies
└── README.md                      # This file
```
