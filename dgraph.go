package main

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	"google.golang.org/grpc"
)

type dgraph struct {
	c *dgo.Dgraph
}

func newDgraph(hosts []string) (*dgraph, error) {
	var cli []api.DgraphClient
	for _, host := range hosts {
		conn, err := grpc.Dial(host, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		cli = append(cli, api.NewDgraphClient(conn))
	}
	return &dgraph{c: dgo.NewDgraphClient(cli...)}, nil
}

func (d *dgraph) setup(ctx context.Context) error {
	return d.c.Alter(ctx, &api.Operation{
		Schema: `
			docKey: string @index(hash) @upsert .
			body: string .
			type Doc {
				docKey: string
				body: string
			}
		`,
	})
}

var randSeed = time.Now().UnixNano()

func (d *dgraph) run(ctx context.Context, size int) (*timingSet, error) {
	var (
		r       = rand.New(rand.NewSource(atomic.AddInt64(&randSeed, 1)))
		t       = &timingSet{}
		tx      = d.c.NewTxn()
		docKey  = randomString(r, 8)
		docBody = randomString(r, size)
	)
	defer tx.Discard(ctx)

	uid, dur, err := d.lookupDoc(ctx, tx, docKey)
	t.Add("lookupDoc", dur)
	if err != nil {
		return nil, err
	}

	uid, dur, err = d.updateDoc(ctx, tx, uid, docKey, docBody)
	t.Add("updateDoc", dur)
	if err != nil {
		return nil, err
	}

	commitDone := t.Start("commit")
	err = tx.Commit(ctx)
	commitDone()
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (d *dgraph) lookupDoc(ctx context.Context, tx *dgo.Txn, key string) (string, time.Duration, error) {
	r, err := tx.QueryWithVars(ctx, `
		query Doc($key: string) {
			q(func: eq(docKey, $key)) {
				uid
				body
			}
		}
	`, map[string]string{"$key": key})
	if err != nil {
		return "", 0, err
	}

	var result struct {
		Q []struct {
			UID string
		}
	}
	if err = json.Unmarshal(r.GetJson(), &result); err != nil {
		return "", 0, err
	}
	if len(result.Q) > 0 {
		return result.Q[0].UID, d.latency(r), nil
	}
	return "", d.latency(r), nil
}

func (d *dgraph) updateDoc(ctx context.Context, tx *dgo.Txn, uid, key, body string) (string, time.Duration, error) {
	data := map[string]interface{}{
		"dgraph.type": "Doc",
		"uid":         uid,
		"docKey":      key,
		"body":        body,
	}
	if uid == "" {
		data["uid"] = "_:new"
	}
	var (
		mu  = &api.Mutation{}
		err error
	)
	mu.SetJson, err = json.Marshal(data)
	if err != nil {
		return "", 0, err
	}

	r, err := tx.Mutate(ctx, mu)
	if err != nil {
		return "", 0, err
	}

	if uid == "" {
		uid = r.GetUids()["new"]
	}
	return uid, d.latency(r), err
}

func (d *dgraph) latency(r *api.Response) time.Duration {
	return time.Duration(r.GetLatency().GetProcessingNs())
}
