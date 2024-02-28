// Copyright (c) 2021 6 River Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.6river.tech/gosix/logging"
	"go.6river.tech/gosix/pubsub"
	"go.6river.tech/gosix/registry"
)

// TODO: add the endpoints for this controller to the OAS spec

type SubscriptionController struct {
	logger *logging.Logger
}

func (sc *SubscriptionController) Register(reg *registry.Registry, router gin.IRouter) error {
	if sc.logger == nil {
		sc.logger = logging.GetLogger("controllers/pubsub/publisher")
	}

	reg.RegisterMap(router, apiRoot+"/subscription", registry.HandlerMap{
		{http.MethodGet, ""}:              sc.GetSubscriptions,
		{http.MethodGet, "/"}:             sc.GetSubscriptions,
		{http.MethodGet, "/:id"}:          sc.GetSubscription,
		{http.MethodGet, "/:id/messages"}: sc.GetMessages,
	})

	return nil
}

func (sc *SubscriptionController) GetSubscriptions(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	i := pubsub.MustDefaultClient().Subscriptions(ctx)
	writeSubscriptions(c, ctx, i)
}

func (sc *SubscriptionController) GetSubscription(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	id := c.Param("id")
	s := pubsub.MustDefaultClient().Subscription(id)
	writeSubscription(c, ctx, s, true)
}

func writeSubscriptions(c *gin.Context, ctx context.Context, i pubsub.SubscriptionIterator) {
	// avoid writing any header until we have some data, so we're more likely to
	// be able to report errors early on
	wroteHeader := false
	// demo streaming JSON info
	for {
		s, err := i.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// let gin try to report it, won't work if we already started writing results
			panic(fmt.Errorf("Error iterating subscriptions: %w", err))
		}
		if !wroteHeader {
			c.Status(http.StatusOK)
			render.JSON{}.WriteContentType(c.Writer)
			// start a json array
			mustWriteString(c, "[\n")
			wroteHeader = true
		} else {
			// not the first one, write a delimiter
			mustWriteString(c, ",\n")
		}
		writeSubscription(c, ctx, s, false)
	}
	if wroteHeader {
		// terminate the JSON array
		mustWriteString(c, "\n]\n")
	} else {
		// write an empty array
		c.JSON(http.StatusOK, []interface{}{})
		_, err := c.Writer.WriteString("\n")
		if err != nil {
			panic(fmt.Errorf("Failed to write JSON delmiter: %w", err))
		}
	}
}

func writeSubscription(c *gin.Context, ctx context.Context, s pubsub.Subscription, writeHeader bool) {
	config, err := s.Config(ctx)
	if err != nil {
		if writeHeader && status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Subscription does not exist",
			})
			return
		}

		panic(fmt.Errorf("Error fetching subscription config for '%s': %w", s.ID(), err))
	}
	topicID := config.Topic.ID()
	// don't write the topic object, it's not meaningful; this won't remove the
	// key, but it hides a lot of object bloat
	config.Topic = nil
	r := render.JSON{Data: gin.H{
		"id":     s.ID(),
		"topic":  topicID,
		"config": config,
	}}
	if writeHeader {
		c.Status(http.StatusOK)
		r.WriteContentType(c.Writer)
	}
	if err = r.Render(c.Writer); err != nil {
		panic(fmt.Errorf("Error serializing subscription info to JSON for '%s': %w", s.ID(), err))
	}
}

func (sc *SubscriptionController) GetMessages(c *gin.Context) {
	id := c.Param("id")
	s := pubsub.MustDefaultClient().Subscription(id)
	// TODO: allow user to specify duration in query param

	// run until either duration expires or we get a message
	duration := 5 * time.Second
	ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
	defer cancel()

	// this is mostly to demo/test this method, but also serves as a check if the
	// subscription doesn't exist
	_, err := s.EnsureDefaultConfig(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Subscription does not exist",
			})
			return
		}

		panic(fmt.Errorf("Error during subscription configuration: %w", err))
	}

	var message pubsub.Message
	mu := &sync.Mutex{}
	err = s.Receive(ctx, func(_ context.Context, m pubsub.Message) {
		// don't try for more messages even if there's time left
		defer cancel()
		mu.Lock()
		defer mu.Unlock()
		// only accept one message
		if message != nil {
			m.Nack()
		} else {
			message = m
			m.Ack()
		}
	})
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			// nothing to worry about here
		} else if status.Code(err) == codes.NotFound {
			// this should have been dealt with above, but ...
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Subscription does not exist",
			})
			return
		} else {
			panic(fmt.Errorf("Error during receive: %w", err))
		}
	}
	if message == nil {
		c.JSON(http.StatusRequestTimeout, gin.H{
			"message": "No message received within timeout",
		})
		return
	}

	mm := message.RealMessage()
	data := mm.Data
	mm.Data = nil
	// try to parse payload as JSON, fall back on reporting it as a string
	var payload interface{}
	if err = json.Unmarshal(data, &payload); err != nil {
		payload = string(data)
	}
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"payload": payload,
	})
}
