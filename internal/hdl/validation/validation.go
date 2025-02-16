package validation

import "github.com/JMURv/golang-clean-template/internal/dto"

func CreateUserRequest(req *dto.CreateUserRequest) error {
	if req.Username == "" {
		return ErrUsernameIsRequired
	}

	if req.Password == "" {
		return ErrPasswordIsRequired
	}
	return nil
}
