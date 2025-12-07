package client

import (
	"bytes"
	"context"

	"github.com/matt-dz/wecook/docs"
)

var _ StrictServerInterface = (*Server)(nil)

type Server struct{}

func NewServer() Server {
	return Server{}
}

func (Server) GetApiPing(ctx context.Context, request GetApiPingRequestObject) (GetApiPingResponseObject, error) {
	return GetApiPing200Response{}, nil
}

func (Server) GetApiOpenapiYaml(
	ctx context.Context,
	request GetApiOpenapiYamlRequestObject,
) (GetApiOpenapiYamlResponseObject, error) {
	data, err := docs.Docs.ReadFile("api.yaml")
	if err != nil {
		return nil, err
	}

	return GetApiOpenapiYaml200ApplicationxYamlResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
	}, nil
}
