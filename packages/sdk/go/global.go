// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
	"net/http"
	"slices"

	"github.com/sst/opencode-sdk-go/internal/apijson"
	"github.com/sst/opencode-sdk-go/internal/requestconfig"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/sst/opencode-sdk-go/packages/respjson"
	"github.com/sst/opencode-sdk-go/packages/ssestream"
)

// GlobalService contains methods and other services that help with interacting
// with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewGlobalService] method instead.
type GlobalService struct {
	Options []option.RequestOption
}

// NewGlobalService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewGlobalService(opts ...option.RequestOption) (r GlobalService) {
	r = GlobalService{}
	r.Options = opts
	return
}

// Get events
func (r *GlobalService) GetEventsStreaming(ctx context.Context, opts ...option.RequestOption) (stream *ssestream.Stream[GlobalGetEventsResponse]) {
	var (
		raw *http.Response
		err error
	)
	opts = slices.Concat(r.Options, opts)
	opts = append([]option.RequestOption{option.WithHeader("Accept", "text/event-stream")}, opts...)
	path := "global/event"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &raw, opts...)
	return ssestream.NewStream[GlobalGetEventsResponse](ssestream.NewDecoder(raw), err)
}

type GlobalGetEventsResponse struct {
	Directory string     `json:"directory,required"`
	Payload   EventUnion `json:"payload,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Directory   respjson.Field
		Payload     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r GlobalGetEventsResponse) RawJSON() string { return r.JSON.raw }
func (r *GlobalGetEventsResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
