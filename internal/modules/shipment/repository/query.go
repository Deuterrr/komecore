package repository

type CalculateCostInput struct {
	OriginID      int
	DestinationID int
	Weight        int
	Couriers      []string
	PriceFilter   *string
}

type CostOption struct {
	Name        string
	Code        string
	Service     string
	Description string
	Cost        int64
	Etd         string
}
