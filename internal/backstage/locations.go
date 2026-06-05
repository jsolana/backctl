package backstage

import (
	"context"
	"fmt"

	"github.com/jsolana/backctl/internal/client"
)

type LocationsService struct {
	client *client.Client
}

func NewLocationsService(c *client.Client) *LocationsService {
	return &LocationsService{client: c}
}

func (s *LocationsService) List(ctx context.Context) ([]LocationEntry, error) {
	var locations []LocationEntry
	err := s.client.GetJSON(ctx, "/api/catalog/locations", nil, &locations)
	return locations, err
}

func (s *LocationsService) GetByID(ctx context.Context, id string) (*LocationEntry, error) {
	path := fmt.Sprintf("/api/catalog/locations/%s", id)
	var location LocationEntry
	err := s.client.GetJSON(ctx, path, nil, &location)
	if err != nil {
		return nil, err
	}
	return &location, nil
}
