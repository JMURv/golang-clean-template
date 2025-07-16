package dto

type DeviceRequest struct {
	IP string `json:"ip"`
	UA string `json:"ua"`
}

type UpdateDeviceRequest struct {
	Name string `json:"name" validate:"required"`
}
