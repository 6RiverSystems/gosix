package pubsub

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientDupe(t *testing.T) {
	ctx := context.Background()
	reg := prometheus.NewPedanticRegistry()
	projectID := t.Name()

	c1, err := NewClient(ctx, projectID, reg, "test", prometheus.Labels{"foo": "bar"})
	require.NoError(t, err, "must initialize first client ok")
	c2, err := NewClient(ctx, projectID, reg, "test", prometheus.Labels{"foo": "bar"})
	require.NoError(t, err, "second client must initialize ok")

	mc1, mc2 := c1.(*monitoredClient), c2.(*monitoredClient)
	assert.Equal(t, mc1.publishStarted, mc2.publishStarted, "publishStarted metric should be reused")
	assert.Equal(t, mc1.published, mc2.published, "publishStarted metric should be reused")
	assert.Equal(t, mc1.publishFailed, mc2.publishFailed, "publishStarted metric should be reused")
	assert.Equal(t, mc1.messagesReceived, mc2.messagesReceived, "publishStarted metric should be reused")
	assert.Equal(t, mc1.messageDuration, mc2.messageDuration, "publishStarted metric should be reused")
}
