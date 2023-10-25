package handlers

import "gopkg.in/go-playground/validator.v9"

var (
	v = validator.New()
)

//ProductValidator a product validator
type ProductValidator struct {
	validator *validator.Validate
}

//Validate validates a product
func (p *ProductValidator) Validate(i interface{}) error {
	return p.validator.Struct(i)
}

type userValidator struct {
	validator *validator.Validate
}

func (u *userValidator) Validate(i interface{}) error {
	return u.validator.Struct(i)
}

type WalletValidator struct {
	validator *validator.Validate
}

func (w *WalletValidator) Validate(i interface{}) error {
	return w.validator.Struct(i)
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
