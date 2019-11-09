# dgraph-perf

Reproducer for Dgraph performance analysis.

First, start the Dgraph cluster. This is based on the [dgraph-ha] Kubernetes
config, but can be done locally as below.


```bash
$ dgraph zero --cwd=z0 --replicas=3 &
$ dgraph zero --cwd=z1 --idx=2 --port_offset=1 --peer=localhost:5080 --replicas=3 &
$ dgraph zero --cwd=z2 --idx=3 --port_offset=2 --peer=localhost:5080 --replicas=3 &
$ dgraph alpha --lru_mb=2048 --zero=localhost:5080 --cwd=a0 &
$ dgraph alpha --lru_mb=2048 --zero=localhost:5080 --cwd=a1 --idx=2 --port_offset=1 &
$ dgraph alpha --lru_mb=2048 --zero=localhost:5080 --cwd=a2 --idx=3 --port_offset=2 &
```

Next, seed with 1 million documents with 10k size. This will take some time and
may need to be restarted if transient errors are encountered.

```bash
$ go run github.com/jfancher/dgraph-perf -size=10000 -threads=100 -count=1000000 -hosts=localhost:9080,localhost:9081,localhost:9082
```

On a reasonably powerful Mac Pro (with SSD), this starts out taking about 200ms
per transaction, creeping up to >1s as it approaches 1 million. Most of the
added time is in commit, although the query also slows down substantially.

Run with a more reasonble level of concurrency to see steady state:
```bash
$ go run github.com/jfancher/dgraph-perf -size=10000 -threads=20  -hosts=localhost:9080,localhost:9081,localhost:9082
```
On the same machine as above, transactions are taking at least 650ms, with
individual items regularly spiking up to 1s+.

[dgraph-ha]: https://github.com/dgraph-io/dgraph/blob/v1.1.0/contrib/config/kubernetes/dgraph-ha/dgraph-ha.yaml
