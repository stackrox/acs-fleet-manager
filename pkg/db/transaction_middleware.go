package db

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"

	serviceError "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
)

// TransactionMiddleware creates a new HTTP middleware that begins a database transaction
// and stores it in the request context.
func TransactionMiddleware(db *ConnectionFactory) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return transactionMiddleware(db, next)
	}
}

func transactionMiddleware(db *ConnectionFactory, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new Context with the transaction stored in it.
		ctx, err := db.NewContext(r.Context())
		if err != nil {
			ulog := logger.NewUHCLogger(ctx)
			ulog.Error(errors.Wrap(err, "Could not create transaction"))
			// use default error to avoid exposing internals to users
			err := serviceError.GeneralError("")
			operationID := logger.GetOperationID(ctx)
			writeJSONResponse(w, err.HttpCode, err.AsOpenapiError(operationID, r.RequestURI))
			return
		}

		// Set the value of the request pointer to the value of a new copy of the request with the new context key,vale stored in it
		*r = *r.WithContext(ctx)

		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			hub.ConfigureScope(func(scope *sentry.Scope) {
				txid := ctx.Value("txid").(int64)
				scope.SetTag("db_transaction_id", fmt.Sprintf("%d", txid))
			})
		}

		// Returned from handlers and resolve transactions.
		defer func() {
			err := Resolve(r.Context())
			if err != nil {
				ulog := logger.NewUHCLogger(ctx)
				ulog.Error(err)
			}
		}()

		// Continue handling requests.
		next.ServeHTTP(w, r)
	})
}

func writeJSONResponse(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		response, _ := json.Marshal(payload)
		_, _ = w.Write(response)
	}
}
