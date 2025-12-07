package client

import (
	"bytes"
	"context"
	"strconv"

	"github.com/matt-dz/wecook/docs"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
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
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	data, err := docs.Docs.ReadFile("api.yaml")
	if err != nil {
		return GetApiOpenapiYaml500JSONResponse{
			Code:    apiError.InternalServerError.String(),
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return GetApiOpenapiYaml200ApplicationxYamlResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
	}, nil
}
