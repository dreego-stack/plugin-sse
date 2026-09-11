package core

import (
	"github.com/dreego-stack/dreego/core/internal/context"
	"github.com/dreego-stack/dreego/core/internal/render"
)

type Result = render.Result

func NewContext() RenderContext {
	return context.NewRender(nil)
}

func Render(component Component) (Result, error) {
	return render.Component(component.Render)
}
