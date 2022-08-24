package promapi

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/prymitive/current"
	"github.com/rs/zerolog/log"
)

type QueryResult struct {
	URI    string
	Series []model.Sample
}

type instantQuery struct {
	prom      *Prometheus
	ctx       context.Context
	expr      string
	timestamp time.Time
}

func (q instantQuery) Run() queryResult {
	log.Debug().
		Str("uri", q.prom.uri).
		Str("query", q.expr).
		Msg("Running prometheus query")

	ctx, cancel := context.WithTimeout(q.ctx, q.prom.timeout)
	defer cancel()

	qr := queryResult{expires: q.timestamp.Add(cacheExpiry * 2)}

	args := url.Values{}
	args.Set("query", q.expr)
	resp, err := q.prom.doRequest(ctx, http.MethodPost, "/api/v1/query", args)
	if err != nil {
		qr.err = err
		return qr
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		qr.err = tryDecodingAPIError(resp)
		return qr
	}

	qr.value, qr.err = streamSamples(resp.Body)
	return qr
}

func (q instantQuery) Endpoint() string {
	return "/api/v1/query"
}

func (q instantQuery) String() string {
	return q.expr
}

func (q instantQuery) CacheKey() string {
	h := sha1.New()
	_, _ = io.WriteString(h, q.Endpoint())
	_, _ = io.WriteString(h, "\n")
	_, _ = io.WriteString(h, q.expr)
	_, _ = io.WriteString(h, "\n")
	_, _ = io.WriteString(h, q.timestamp.Round(cacheExpiry).Format(time.RFC3339))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (p *Prometheus) Query(ctx context.Context, expr string) (*QueryResult, error) {
	log.Debug().Str("uri", p.uri).Str("query", expr).Msg("Scheduling prometheus query")

	key := fmt.Sprintf("/api/v1/query/%s", expr)
	p.locker.lock(key)
	defer p.locker.unlock(key)

	resultChan := make(chan queryResult)
	p.queries <- queryRequest{
		query:  instantQuery{prom: p, ctx: ctx, expr: expr, timestamp: time.Now()},
		result: resultChan,
	}

	result := <-resultChan
	if result.err != nil {
		return nil, QueryError{err: result.err, msg: decodeError(result.err)}
	}

	qr := QueryResult{
		URI:    p.uri,
		Series: result.value.([]model.Sample),
	}
	log.Debug().Str("uri", p.uri).Str("query", expr).Int("series", len(qr.Series)).Msg("Parsed response")

	return &qr, nil
}

func streamSamples(r io.Reader) (samples []model.Sample, err error) {
	defer dummyReadAll(r)

	var status, resultType, errType, errText string
	samples = []model.Sample{}
	decoder := current.Object(
		func() {},
		current.Key("status", current.Text(func(s string) {
			status = s
		})),
		current.Key("error", current.Text(func(s string) {
			errText = s
		})),
		current.Key("errorType", current.Text(func(s string) {
			errType = s
		})),
		current.Key("data", current.Object(
			func() {},
			current.Key("resultType", current.Text(func(s string) {
				resultType = s
			})),
			current.Key("result", current.Array(func(sample *model.Sample) {
				samples = append(samples, *sample)
				sample.Metric = model.Metric{}
			})),
		)),
	)

	dec := json.NewDecoder(r)
	if err = current.Stream(dec, decoder); err != nil {
		return nil, APIError{Status: status, ErrorType: v1.ErrBadResponse, Err: fmt.Sprintf("JSON parse error: %s", err)}
	}

	if status != "success" {
		return nil, APIError{Status: status, ErrorType: decodeErrorType(errType), Err: errText}
	}

	if resultType != "vector" {
		return nil, APIError{Status: status, ErrorType: v1.ErrBadResponse, Err: fmt.Sprintf("invalid result type, expected vector, got %s", resultType)}
	}

	return samples, nil
}
