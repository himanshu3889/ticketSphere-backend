package utils

import (
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// safely ends a span and records any errors automatically to otel trace.
func EndSpan(span trace.Span, err error) {
	if err != nil {
		// span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

// safely ends a span and records any errors automatically to otel trace for appError type.
func EndSpanAppErr(span trace.Span, err *appError.Error) {
	if err != nil {
		internalMsg := ""
		if err.InternalMessage != nil {
			internalMsg = *err.InternalMessage
		}
		// span.RecordError(errors.New(internalMsg))
		span.SetStatus(codes.Error, internalMsg)
	}
	span.End()
}
