package auth

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/JMURv/golang-clean-template/internal/dto"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/mssola/useragent"
)

func GenerateDevice(d *dto.DeviceRequest) md.Device {
	hash := sha256.Sum256([]byte(d.IP + d.UA))
	ua := useragent.New(d.UA)
	bName, _ := ua.Browser()

	var dt string

	switch {
	case ua.Mobile():
		dt = "mobile"
	case ua.Bot():
		dt = "bot"
	case ua.Mozilla() != "":
		dt = "desktop"
	case !ua.Mobile() && !ua.Bot() && ua.Mozilla() == "":
		dt = "tablet"
	default:
		dt = "unknown"
	}

	return md.Device{
		ID:         hex.EncodeToString(hash[:8]),
		Name:       "My " + dt,
		DeviceType: dt,
		OS:         ua.OS(),
		Browser:    bName,
		UA:         d.UA,
		IP:         d.IP,
	}
}
