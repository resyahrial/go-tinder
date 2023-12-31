package rest

import (
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/pkg/errors"
)

type bindValidator struct {
	once       sync.Once
	validate   *validator.Validate
	translator ut.Translator
}

var _ binding.StructValidator = &bindValidator{}

func (v *bindValidator) ValidateStruct(obj any) error {
	v.lazyinit()
	err := v.validate.Struct(obj)
	if err == nil {
		return nil
	}
	validatorErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}
	for _, e := range validatorErrs {
		message := e.Translate(v.translator)
		return errors.New(message)
	}
	return err
}

func (v *bindValidator) Engine() any {
	v.lazyinit()
	return v.validate
}

func (v *bindValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		english := en.New()
		uni := ut.New(english, english)
		trans, ok := uni.GetTranslator("en")
		if !ok {
			panic(errors.New("failed to get translator"))
		}
		v.translator = trans
		if err := en_translations.RegisterDefaultTranslations(v.validate, v.translator); err != nil {
			panic(errors.Wrap(err, "failed to register translator"))
		}
	})
}
