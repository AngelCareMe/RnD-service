package cbr

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *logrus.Logger
}

func NewClient(logger *logrus.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://www.cbr.ru/scripts",
		logger:  logger,
	}
}

func (c *Client) FetchRates(ctx context.Context, date string) (*ValCurs, error) {
	url := fmt.Sprintf("%s/XML_daily.asp?date_req=%s", c.baseURL, date)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.logger.Errorf("Failed to fetch by API")
		return nil, fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Errorf("Bad status response API")
		return nil, fmt.Errorf("bad status: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Errorf("Failed to read response body by API")
		return nil, fmt.Errorf("error read response body: %w", err)
	}

	var valCurs ValCurs
	if err := xml.Unmarshal(body, &valCurs); err != nil {
		c.logger.Errorf("Failed to parse XML CBR")
		return nil, fmt.Errorf("error parse XML: %w", err)
	}

	return &valCurs, nil
}
