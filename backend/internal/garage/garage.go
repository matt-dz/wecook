// Package garage contains utility functions for garage.
package garage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/go-retryablehttp"
	mHttp "github.com/matt-dz/wecook/internal/http"
	mJson "github.com/matt-dz/wecook/internal/json"
)

type APIClient struct {
	*garage.APIClient

	AdminToken string
}

type role struct {
	Zone     *string   `json:"zone"`
	Tags     *[]string `json:"tags"`
	Capacity *int64    `json:"capacity"`
	ID       *string   `json:"id"`
}

type partition struct {
	Available *int64 `json:"available"`
	Total     *int64 `json:"total"`
}

type node struct {
	ID                string     `json:"id"`
	GarageVersion     *string    `json:"garageVersion"`
	Addr              *string    `json:"addr"`
	Hostname          *string    `json:"hostname"`
	IsUp              *bool      `json:"isUp"`
	LastSeenSecsAgo   *int64     `json:"lastSeenSecsAgo"`
	Draining          *bool      `json:"draining"`
	DataPartition     *partition `json:"dataPartition"`
	MetadataPartition *partition `json:"metadataPartition"`
}

type getClusterStatusResponse struct {
	LayoutVersion *int    `json:"layoutVersion"`
	Nodes         *[]node `json:"nodes"`
}

type zoneRedundancy struct {
	AtLeast int64 `json:"atLeast"`
}

type layoutParameters struct {
	ZoneRedundancy zoneRedundancy `json:"zoneRedundancy"`
}

type updateClusterLayoutRequest struct {
	Parameters layoutParameters `json:"parameters"`
	Roles      []role           `json:"roles"`
}

type applyClusterLayoutRequest struct {
	Version int64 `json:"version"`
}

var (
	WeCookImagesBucket = "wecook-images"
	layoutTags         = []string{"storage"}
)

const (
	defaultZone           = "dc1"
	defaultTag            = "storage"
	defaultCapacity int64 = 500_000_000_000 // 500 GB
)

func newUpdateClusterLayoutRequest(atLeast int64) updateClusterLayoutRequest {
	return updateClusterLayoutRequest{
		Parameters: layoutParameters{
			zoneRedundancy{
				AtLeast: atLeast,
			},
		},
		Roles: []role{},
	}
}

func NewClient(adminHost, adminToken string, httpClient *mHttp.HTTP) (*APIClient, error) {
	config := garage.NewConfiguration()

	// Set host
	config.Host = adminHost
	config.HTTPClient = httpClient.HTTPClient
	client := garage.NewAPIClient(config)

	// Check admin token
	return &APIClient{
		APIClient:  client,
		AdminToken: adminToken,
	}, nil
}

func InitializeGarage(http *mHttp.HTTP, ctx context.Context, adminHost, apiToken string) error {
	// Get cluster status
	req, err := retryablehttp.NewRequestWithContext(ctx,
		"GET", fmt.Sprintf("http://%s/v2/GetClusterStatus", adminHost), nil)
	if err != nil {
		return fmt.Errorf("creating get cluster status request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiToken))
	resp, err := http.Do(req)
	if err != nil {
		return fmt.Errorf("getting cluster status: %w", err)
	}
	if err := mHttp.ExpectStatus2xx(resp); err != nil {
		return fmt.Errorf("failed to get cluster status: %w", err)
	}
	var clusterStatus getClusterStatusResponse
	decoder := json.NewDecoder(resp.Body)
	defer func() { _ = resp.Body.Close }()
	if err := mJson.DecodeJSON(&clusterStatus, decoder); err != nil {
		return fmt.Errorf("decoding cluster status response: %w", err)
	}
	var layoutVersion int
	if clusterStatus.LayoutVersion != nil {
		layoutVersion = *clusterStatus.LayoutVersion
	}
	if layoutVersion > 0 {
		return nil // layout already applied, no need to continue
	}

	// Update cluster layout
	clusterLayoutReq := newUpdateClusterLayoutRequest(1)
	if clusterStatus.Nodes == nil || len(*clusterStatus.Nodes) == 0 {
		return errors.New("no nodes found in garage cluster")
	}
	for _, node := range *clusterStatus.Nodes {
		zone := defaultZone
		capacity := defaultCapacity
		clusterLayoutReq.Roles = append(clusterLayoutReq.Roles, role{
			Zone:     &zone,
			Capacity: &capacity,
			Tags:     &layoutTags,
			ID:       &node.ID,
		})
	}
	body, err := json.Marshal(clusterLayoutReq)
	if err != nil {
		return fmt.Errorf("marshaling update cluster layout request: %w", err)
	}
	req, err = retryablehttp.NewRequestWithContext(ctx,
		"POST", fmt.Sprintf("http://%s/v2/UpdateClusterLayout", adminHost), body)
	if err != nil {
		return fmt.Errorf("creating update cluster layout request: %w", err)
	}
	if err := mHttp.ExpectStatus2xx(resp); err != nil {
		return fmt.Errorf("failed to update cluster layout: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiToken))
	_, err = http.Do(req)
	if err != nil {
		return fmt.Errorf("updating cluster layout: %w", err)
	}

	// Apply cluster layout
	applyLayoutReq := applyClusterLayoutRequest{
		Version: int64(layoutVersion + 1),
	}
	body, err = json.Marshal(applyLayoutReq)
	if err != nil {
		return fmt.Errorf("marshaling applying cluster layout request: %w", err)
	}
	req, err = retryablehttp.NewRequestWithContext(ctx,
		"POST", fmt.Sprintf("http://%s/v2/ApplyClusterLayout", adminHost), body)
	if err != nil {
		return fmt.Errorf("creating apply cluster layout request: %w", err)
	}
	if err := mHttp.ExpectStatus2xx(resp); err != nil {
		return fmt.Errorf("failed to apply cluster layout: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiToken))
	if _, err = http.Do(req); err != nil {
		return fmt.Errorf("applying cluster layout: %w", err)
	}

	return nil
}
