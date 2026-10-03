package http

type createInventoryRequest struct {
	Stock int `json:"stock"`
}

type updateInventoryRequest struct {
	Stock int `json:"stock"`
}
