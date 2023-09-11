package entity

type PaginationParam struct {
	PageSize int
	Offset   int
	Search   string
	SortBy   string
	IsActive bool
	//Refresh  bool
}

type PaginationResponse struct {
	Total           int
	HasNextPage     bool
	HasPreviousPage bool
	Data            []*Asset
}
