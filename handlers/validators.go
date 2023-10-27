package handlers

import "gopkg.in/go-playground/validator.v9"

var (
	v = validator.New()
)

type userValidator struct {
	validator *validator.Validate
}

func (u *userValidator) Validate(i interface{}) error {
	return u.validator.Struct(i)
}


type rewardValidator struct {
	validator *validator.Validate
}

func (v *rewardValidator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}

type userRewardValidator struct {
	validator *validator.Validate
}

func (ur *userRewardValidator) Validate(i interface{}) error {
	return ur.validator.Struct(i)
}
