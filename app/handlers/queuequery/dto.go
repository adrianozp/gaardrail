package handlers

type setQueueQueryRequest struct {
	Query string `json:"query" binding:"required"`
}

type queueQueryResponse struct {
	Query string `json:"query"`
}
