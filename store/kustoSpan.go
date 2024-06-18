package store

import (
	"fmt"

	"encoding/json"
	"strconv"
	"time"

	"github.com/dodopizza/jaeger-kusto/store/utils"

	"github.com/Azure/azure-kusto-go/kusto/data/value"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type kustoSpan struct {
	TraceID            string        `kusto:"TraceID"`
	SpanID             string        `kusto:"SpanID"`
	ParentID           string        `kusto:"ParentID"`
	SpanName           string        `kusto:"SpanName"`
	SpanStatus         string        `kusto:"SpanStatus"`
	SpanKind           string        `kusto:"SpanKind"`
	StartTime          time.Time     `kusto:"StartTime"`
	EndTime            time.Time     `kusto:"EndTime"`
	ResourceAttributes value.Dynamic `kusto:"ResourceAttributes"`
	TraceAttributes    value.Dynamic `kusto:"TraceAttributes"`
	Events             value.Dynamic `kusto:"Events"`
	Links              value.Dynamic `kusto:"Links"`
}

type Link struct {
	TraceID            string
	SpanID             string
	TraceState         string
	SpanLinkAttributes map[string]any
}

const (
	// TagDotReplacementCharacter state which character should replace the dot in dynamic column
	TagDotReplacementCharacter = "_"
)

func transformKustoSpanToModelSpan(kustoSpan *kustoSpan) (*model.Span, error) {
	fmt.Printf("getting row 3 %s!!!!\n", kustoSpan.TraceID)

	/*internalResource, internalSpan, err := kustoSpanToOtelInternalSpan(kustoSpan)
	if err != nil {
		return nil, err
	}

	batches, err := jaeger.


	spanConverter := dbmodel.NewToDomain(TagDotReplacementCharacter)
	convertedSpan, err := spanConverter.SpanToDomain(jsonSpan)
	if err != nil {
		return nil, err
	}*/

	span := &model.Span{
		/*TraceID:       convertedSpan.TraceID,
		SpanID:        convertedSpan.SpanID,
		OperationName: kustoSpan.OperationName,
		References:    convertedSpan.References,
		Flags:         convertedSpan.Flags,
		StartTime:     kustoSpan.StartTime,
		Duration:      kustoSpan.Duration,
		Tags:          convertedSpan.Tags,
		Logs:          convertedSpan.Logs,
		Process:       convertedSpan.Process,*/
	}

	return span, nil
}

func getTagsValues(tags []model.KeyValue) []string {
	var values []string
	for i := range tags {
		values = append(values, tags[i].VStr)
	}
	return values
}

// TransformSpanToStringArray converts span to string ready for Kusto ingestion
func TransformSpanToStringArray(span *model.Span) ([]string, error) {

	spanConverter := dbmodel.NewFromDomain(true, getTagsValues(span.Tags), TagDotReplacementCharacter)
	jsonSpan := spanConverter.FromDomainEmbedProcess(span)

	references, err := json.Marshal(jsonSpan.References)
	if err != nil {
		return nil, err
	}
	tags, err := json.Marshal(jsonSpan.Tag)
	if err != nil {
		return nil, err
	}
	logs, err := json.Marshal(jsonSpan.Logs)
	if err != nil {
		return nil, err
	}
	processTags, err := json.Marshal(jsonSpan.Process.Tag)
	if err != nil {
		return nil, err
	}

	kustoStringSpan := []string{
		span.TraceID.String(),
		span.SpanID.String(),
		span.OperationName,
		string(references),
		strconv.FormatUint(uint64(span.Flags), 10),
		span.StartTime.Format(time.RFC3339Nano),
		value.Timespan{Value: span.Duration, Valid: true}.Marshal(),
		string(tags),
		string(logs),
		span.Process.ServiceName,
		string(processTags),
		span.ProcessID,
	}

	return kustoStringSpan, err
}

func kustoSpanToOtelInternalSpan(kustoSpan *kustoSpan) (pcommon.Resource, ptrace.Span, error) {
	var err error

	internalResource := pcommon.NewResource()
	internalSpan := ptrace.NewSpan()

	var traceId model.TraceID
	var spanId model.SpanID
	var parentSpanId model.SpanID

	// convert kusto trace-id to jaeger trace-id to pcommon.TraceID
	if traceId, err = model.TraceIDFromString(kustoSpan.TraceID); err != nil {
		return internalResource, internalSpan, err
	}

	if spanId, err = model.SpanIDFromString(kustoSpan.SpanID); err != nil {
		return internalResource, internalSpan, err
	}

	if parentSpanId, err = model.SpanIDFromString(kustoSpan.ParentID); err != nil {
		return internalResource, internalSpan, err
	}

	var traceAttributes map[string]interface{}
	if err = json.Unmarshal(kustoSpan.TraceAttributes.Value, &traceAttributes); err != nil {
		return internalResource, internalSpan, err
	}

	var links []Link
	if err = json.Unmarshal(kustoSpan.Links.Value, &links); err != nil {
		return internalResource, internalSpan, err
	}

	internalSpan.SetTraceID(utils.UInt64ToTraceID(traceId.High, traceId.Low))
	internalSpan.SetSpanID(utils.UInt64ToSpanID(uint64(spanId)))
	internalSpan.SetParentSpanID(utils.UInt64ToSpanID(uint64(parentSpanId)))
	internalSpan.SetName(kustoSpan.SpanName)
	internalSpan.SetKind(SpanKindFromStr(kustoSpan.SpanKind))
	internalSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(kustoSpan.StartTime))
	internalSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(kustoSpan.EndTime))
	internalSpan.Attributes().FromRaw(traceAttributes)
	getLinks(links, &internalSpan)

	var resourceAttributes map[string]interface{}
	if err = json.Unmarshal(kustoSpan.ResourceAttributes.Value, &resourceAttributes); err != nil {
		return internalResource, internalSpan, err
	}

	internalResource.Attributes().FromRaw(resourceAttributes)

	return internalResource, internalSpan, nil
}

func SpanKindFromStr(sk string) ptrace.SpanKind {
	switch sk {
	case "SPAN_KIND_INTERNAL":
		return ptrace.SpanKindInternal
	case "SPAN_KIND_SERVER":
		return ptrace.SpanKindServer
	case "SPAN_KIND_CLIENT":
		return ptrace.SpanKindClient
	case "SPAN_KIND_PRODUCER":
		return ptrace.SpanKindProducer
	case "SPAN_KIND_CONSUMER":
		return ptrace.SpanKindConsumer
	}

	return ptrace.SpanKindUnspecified
}

func getLinks(links []Link, span *ptrace.Span) {
	linkSplice := ptrace.NewSpanLinkSlice()
	linkSplice.EnsureCapacity(len(links))

	for i := 0; i < len(links); i++ {
		link := linkSplice.AppendEmpty()

		var traceId model.TraceID
		var spanId model.SpanID
		var err error

		// convert kusto trace-id to jaeger trace-id to pcommon.TraceID
		if traceId, err = model.TraceIDFromString(links[i].TraceID); err != nil {
			return
		}

		if spanId, err = model.SpanIDFromString(links[i].SpanID); err != nil {
			return
		}

		link.SetTraceID(utils.UInt64ToTraceID(traceId.High, traceId.Low))
		link.SetSpanID(utils.UInt64ToSpanID(uint64(spanId)))

		attrs := link.Attributes()
		attrs.FromRaw(links[i].SpanLinkAttributes)
		link.TraceState().FromRaw(getTraceStateFromAttrs(attrs))
	}

	linkSplice.CopyTo(span.Links())
}

func getTraceStateFromAttrs(attrs pcommon.Map) string {
	traceState := ""
	traceStateKey := "w3c.tracestate"
	// TODO Bring this inline with solution for jaegertracing/jaeger-client-java #702 once available
	if attr, ok := attrs.Get(traceStateKey); ok {
		traceState = attr.Str()
		attrs.Remove(traceStateKey)
	}
	return traceState
}
