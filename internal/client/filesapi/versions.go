// CLI source: cli/internal/drapi/filesapi/versions.go
//
// Provider differences from CLI:
// - ListVersions takes context.Context.
// - Pagination uses c.getJSON/c.assertNextOnSameHost instead of drapi.GetJSON/drapi.AssertNextOnSameHost.
// - Initial URL built via c.endpointURL; CLI uses drapi.EndpointURL with separate error return.
package filesapi

import (
	"context"
	"net/url"
	"strconv"
)

const versionsPageSize = 100

func (c *httpClient) ListVersions(ctx context.Context, catalogID string, limit int) ([]CatalogVersion, error) {
	q := url.Values{}
	q.Set("orderBy", "-created")
	q.Set("limit", strconv.Itoa(versionsPageSize))

	// CLI: drapi.EndpointURL("/files/"+catalogID+"/versions/", q)
	pageURL := c.endpointURL("/files/"+url.PathEscape(catalogID)+"/versions/", q)

	out := make([]CatalogVersion, 0)

	for pageURL != "" {
		var page CatalogVersionsResp
		// CLI: drapi.GetJSON(pageURL, "versions", &page)
		if err := c.getJSON(ctx, pageURL, &page); err != nil {
			return nil, err
		}

		for _, v := range page.Data {
			out = append(out, v)
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}

		if page.Next == "" {
			break
		}

		// CLI: drapi.AssertNextOnSameHost(page.Next)
		if err := c.assertNextOnSameHost(page.Next); err != nil {
			return nil, err
		}

		pageURL = page.Next
	}

	return out, nil
}
