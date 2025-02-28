// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hooks

import (
	"context"
	"sync"
	"testing"
	"time"

	fnpb "github.com/apache/beam/sdks/go/pkg/beam/model/fnexecution_v1"
)

type contextKey string

const (
	initKey   = contextKey("init_key")
	reqKey    = contextKey("req_key")
	initValue = "initValue"
	reqValue  = "reqValue"
)

func initializeHooks() *registry {
	var r = newRegistry()
	r.activeHooks["test"] = Hook{
		Init: func(ctx context.Context) (context.Context, error) {
			return context.WithValue(ctx, initKey, initValue), nil
		},
		Req: func(ctx context.Context, req *fnpb.InstructionRequest) (context.Context, error) {
			return context.WithValue(ctx, reqKey, reqValue), nil
		},
	}
	return r
}

func TestInitContextPropagation(t *testing.T) {
	r := initializeHooks()
	ctx := context.Background()
	var err error

	expected := initValue
	ctx, err = r.RunInitHooks(ctx)
	if err != nil {
		t.Errorf("got %v error, wanted no error", err)
	}
	actual := ctx.Value(initKey)
	if actual != expected {
		t.Errorf("Got %s, wanted %s", actual, expected)
	}
}

func TestRequestContextPropagation(t *testing.T) {
	r := initializeHooks()
	ctx := context.Background()

	expected := reqValue
	ctx = r.RunRequestHooks(ctx, nil)
	actual := ctx.Value(reqKey)
	if actual != expected {
		t.Errorf("Got %s, wanted %s", actual, expected)
	}
}

// TestConcurrentWrites tests if the concurrent writes are handled properly.
// It uses go routines to test this on sample hook 'google_logging'.
func TestConcurrentWrites(t *testing.T) {
	r := initializeHooks()
	hf := func(opts []string) Hook {
		return Hook{
			Req: func(ctx context.Context, req *fnpb.InstructionRequest) (context.Context, error) {
				return ctx, nil
			},
		}
	}
	r.RegisterHook("google_logging", hf)

	var actual, expected error
	expected = nil

	ch := make(chan struct{})
	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ch:
					// When the channel is closed, exit.
					return
				default:
					actual = r.EnableHook("google_logging")
					if actual != expected {
						t.Errorf("Got %s, wanted %s", actual, expected)
					}
				}
			}
		}()
	}
	// Let the goroutines execute for 5 seconds and then close the channel.
	time.Sleep(time.Second * 5)
	close(ch)
	// Wait for all goroutines to exit properly.
	wg.Wait()
}
