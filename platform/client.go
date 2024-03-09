// Copyright (C) 2024 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package platform provides functionality for sending telemetry data to Percona Platform.
package platform

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	genericv1 "github.com/percona-platform/saas/gen/telemetry/generic"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/percona-platform/saas/pkg/logger"
)

// ‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger interface
// _______________________________________________________________________

// Logger interface is to abstract the logging from Client. Gives control to
// the Client users, choice of the logger.
type Logger interface {
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}

// Option is an option for Client returned by constructor.
type Option func(*Client)

// WithLogFullRequest enable/disables logging of request/response body and headers.
func WithLogFullRequest() Option {
	return func(c *Client) {
		tr, _ := c.restyClient.Transport()
		c.restyClient.SetTransport(
			logger.HTTP(tr, "perconaPlatformClient", logger.LogFullRequest()),
		)
	}
}

// WithResendTimeout method sets default wait time to sleep before retrying
// request.
//
// Default is 100 milliseconds.
func WithResendTimeout(resendTimeout time.Duration) Option {
	return func(c *Client) {
		c.restyClient.SetRetryWaitTime(resendTimeout).
			AddRetryCondition(
				func(r *resty.Response, err error) bool {
					return r.StatusCode() == http.StatusRequestTimeout ||
						r.StatusCode() >= http.StatusInternalServerError
				},
			)
	}
}

// WithRetryCount method enables retry on client and allows you
// to set no. of retry count. Client uses a Backoff mechanism.
func WithRetryCount(count int) Option {
	return func(c *Client) {
		c.restyClient.SetRetryCount(count)
	}
}

// WithLogger method sets given writer for logging client request and response details.
//
// Compliant to interface `Logger`.
func WithLogger(l Logger) Option {
	return func(c *Client) {
		c.restyClient.SetLogger(l)
	}
}

// WithBaseURL method is to set Base URL in the client instance. It will be used with request
// raised from this client with relative URL
//
//	// Setting HTTP address
//	client.WithBaseURL("http://myjeeva.com")
//
//	// Setting HTTPS address
//	client.WithBaseURL("https://myjeeva.com").
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.restyClient.SetBaseURL(url)
	}
}

// WithTLSClientConfig method sets TLSClientConfig for underling client Transport.
//
// For Example:
//
//	// One can set custom root-certificate. Refer: http://golang.org/pkg/crypto/tls/#example_Dial
//	client.WithTLSClientConfig(&tls.Config{ RootCAs: roots })
//
//	// or One can disable security check (https)
//	client.WithTLSClientConfig(&tls.Config{ InsecureSkipVerify: true })
//
// Note: This method overwrites existing `TLSClientConfig`.
func WithTLSClientConfig(config *tls.Config) Option {
	return func(c *Client) {
		c.restyClient.SetTLSClientConfig(config)
	}
}

// WithClientTimeout method sets timeout for request raised from client.
//
// client.WithClientTimeout(time.Duration(1 * time.Minute)).
func WithClientTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.restyClient.SetTimeout(timeout)
	}
}

// Client is HTTP Percona Platform client.
type Client struct {
	restyClient *resty.Client
}

// New creates new Percona Platform Telemetry client.
func New(opts ...Option) *Client {
	c := &Client{
		restyClient: resty.New().
			SetContentLength(true).
			SetCloseConnection(false),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// SendTelemetry sends telemetry data to Percona Platform.
func (c *Client) SendTelemetry(ctx context.Context, accessToken string, report *genericv1.ReportRequest) error {
	const path = "/v1/telemetry/GenericReport"

	body, err := protojson.Marshal(report)
	if err != nil {
		return err
	}

	err = c.sendPostRequest(ctx, path, accessToken, bytes.NewReader(body), nil)
	if err != nil {
		return fmt.Errorf("failed to send telemetry data: %w", err)
	}

	return nil
}

// Error is a model of an error response from Percona Platform.
type Error struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details"`
}

// Error error interface implementation.
func (e Error) Error() string {
	return e.String()
}

// String returns a string representation of an error.
func (e Error) String() string {
	parts := make(map[string]string)

	if e.Code > 0 {
		parts["code"] = strconv.Itoa(e.Code)
	}

	if len(e.Message) > 0 {
		parts["message"] = e.Message
	}

	if len(e.Details) != 0 {
		parts["details"] = strings.Join(e.Details, ",")
	}
	return fmt.Sprintf("%v", parts)
}

func (c *Client) sendPostRequest(ctx context.Context, path, accessToken string, requestBody, responseBody interface{}) error {
	req := c.createRequest(ctx)

	if requestBody != nil {
		req.SetBody(requestBody)
	}

	if responseBody != nil {
		// set object for parsing body from response
		req = req.SetResult(responseBody)
	}

	if len(accessToken) > 0 {
		req.SetAuthScheme("Bearer")
		req.SetAuthToken(accessToken)
	}

	resp, err := req.Post(path)

	return checkForError(resp, err)
}

func (c *Client) createRequest(ctx context.Context) *resty.Request {
	var err Error
	req := c.restyClient.R().
		SetContext(ctx).
		SetError(&err)
	req.Header.Add("Accept", "application/json")
	return req
}

func checkForError(resp *resty.Response, err error) error {
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if resp == nil {
		return errors.New("no response from Percona Platform")
	}

	if resp.IsError() {
		if e, ok := resp.Error().(*Error); ok {
			return e
		}
		return errors.New(resp.Status())
	}

	return nil
}
