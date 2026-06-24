package handlers

type switchQueueRequest struct {
	Type string `json:"type" binding:"required"`
}

type switchQueueResponse struct {
	Type      string   `json:"type"`
	Available []string `json:"available"`
}
