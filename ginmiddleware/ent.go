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

package ginmiddleware

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/logging"
)

var (
	entClientKeyBase = "ent-client-"
	entTxKeyBase     = "ent-tx-"
)

func WithEntClient(client ent.EntClient, name string) gin.HandlerFunc {
	key := entClientKeyBase + name
	return func(c *gin.Context) {
		c.Set(key, client)
	}
}

func Client(c *gin.Context, name string) ent.EntClient {
	// TODO: could have this check for an active transaction and return the
	// transactional client instead in that case
	return c.MustGet(entClientKeyBase + name).(ent.EntClient)
}

type TransactionControl func(*gin.Context, *sql.TxOptions) bool

func WithTransaction(
	name string,
	opts *sql.TxOptions,
	controls ...TransactionControl,
) gin.HandlerFunc {
	txKey := entTxKeyBase + name
	logger := logging.GetLogger("middleware/ent")
	if opts == nil {
		opts = &sql.TxOptions{}
	}
	return func(c *gin.Context) {
		client := Client(c, name)
		// make a copy before we mutate it
		txOpts := *opts
		useTx := true
		for _, control := range controls {
			if !control(c, &txOpts) {
				useTx = false
				break
			}
		}
		if !useTx {
			// don't actually want a transaction, move on to the next handler
			return
		}

		tx, err := client.BeginTxGeneric(c.Request.Context(), &txOpts)
		if err != nil {
			// TODO: avoid relying on gin's panic handling
			panic(err)
		}
		c.Set(txKey, tx)
		// TODO: not sure this panic handling is correct
		defer func() {
			// if tx is non-nil, we must have panicked
			if tx != nil {
				rbErr := tx.Rollback()
				if rbErr != nil {
					// nolint:errcheck // return value here is just a wrapped copy of the input
					c.Error(rbErr)
					// we're about to re-panic, don't overwrite the original
					logger.Err(rbErr).Msg("Failed to rollback during panic")
				}
				tx = nil
			}
		}()
		c.Next()
		if len(c.Errors) > 0 || c.IsAborted() {
			rbErr := tx.Rollback()
			if rbErr != nil {
				// nolint:errcheck // return value here is just a wrapped copy of the input
				c.Error(rbErr)
			}
		} else {
			cErr := tx.Commit()
			if cErr != nil {
				// nolint:errcheck // return value here is just a wrapped copy of the input
				c.Error(cErr)
			}
		}
		tx = nil
	}
}

func Transaction(c *gin.Context, name string) ent.EntTx {
	return c.MustGet(entTxKeyBase + name).(ent.EntTx)
}
