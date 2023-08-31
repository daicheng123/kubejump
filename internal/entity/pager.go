package entity

type PaginationParam struct {
	PageSize int
	Offset   int
	Searches map[string]interface{}
	SortBy   string
	//Refresh  bool
}

type PaginationResponse struct {
	Total       int
	NextURL     string
	PreviousURL string
	Data        []Asset
}
