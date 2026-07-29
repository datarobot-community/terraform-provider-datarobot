package client

import (
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// FilesAPI returns a Files API client sharing auth and base URL with this service.
func (s *ServiceImpl) FilesAPI() filesapi.Client {
	return filesapi.New(newFilesAPITransport(s.client))
}
