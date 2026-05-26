package handlers

type switchControllerRequest struct {
	Type string `json:"type" binding:"required"`
}
