package domain

type AddressDetail struct {
	Province    string
	City        string
	District    string
	PostalCode  string
	FullAddress string

	Latitude  *float64
	Longitude *float64
}

func (d AddressDetail) ValidateCoordinates() error {
	if d.Latitude != nil {
		if *d.Latitude < -90.0 || *d.Latitude > 90.0 {
			return ErrInvalidCoordinates
		}
	}
	if d.Longitude != nil {
		if *d.Longitude < -180.0 || *d.Longitude > 180.0 {
			return ErrInvalidCoordinates
		}
	}
	return nil
}
