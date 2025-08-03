package cbr

import "context"

type CbrClient interface {
	FetchRates(ctx context.Context, date string) (*ValCurs, error)
}
